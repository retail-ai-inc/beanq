//go:build integration || ci
// +build integration ci

package bredis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
)

func TestSequenceQueueMultipleConsumersSerializeEachCustomer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, prefix := newSequenceQueueIntegrationClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	prefix = fmt.Sprintf("%ssq_multi_%d", prefix, time.Now().UnixNano())
	topology := newSequenceQueueTopology(prefix, "integration", "multiple-consumers", 2)
	store := newSequenceQueueStore(client, topology, 100, 2*time.Second)
	t.Cleanup(func() { cleanupSequenceQueueIntegrationKeys(t, context.Background(), client, prefix) })

	const group = "multiple-consumer-workers"
	if err := store.ensureMetadata(ctx); err != nil {
		t.Fatalf("ensure metadata: %v", err)
	}
	if err := store.bootstrapGroups(ctx, group); err != nil {
		t.Fatalf("bootstrap groups: %v", err)
	}

	messageCounts := map[string]int{"order-a": 5, "order-b": 4, "order-c": 3}
	total := 0
	for sequence := 1; sequence <= 5; sequence++ {
		for _, orderKey := range []string{"order-a", "order-b", "order-c"} {
			if sequence > messageCounts[orderKey] {
				continue
			}
			messageID := fmt.Sprintf("%s-%02d", orderKey, sequence)
			result, err := store.enqueue(ctx, orderKey, map[string]any{
				"id": messageID, "orderKey": orderKey, "sequence": sequence,
			})
			if err != nil {
				t.Fatalf("publish %s: %v", messageID, err)
			}
			total++
			t.Logf("PUBLISH orderKey=%s sequence=%02d id=%s result=%s", orderKey, sequence, messageID, result.Code)
		}
	}

	type consumption struct {
		orderKey  string
		consumer  string
		messageID string
		sequence  int
		started   time.Time
		finished  time.Time
	}
	var (
		completed atomic.Int64
		mutex     sync.Mutex
		active    = make(map[string]int)
		overlaps  = make(map[string]int)
		records   []consumption
		workers   sync.WaitGroup
	)

	for consumerNumber := 1; consumerNumber <= 4; consumerNumber++ {
		consumer := fmt.Sprintf("consumer-%d", consumerNumber)
		workers.Go(func() {
			for ctx.Err() == nil && completed.Load() < int64(total) {
				processed := false
				for partition := int64(0); partition < topology.partitions; partition++ {
					token, err := store.readOne(ctx, group, consumer, partition, 100*time.Millisecond)
					if errors.Is(err, redis.Nil) || errors.Is(err, context.DeadlineExceeded) {
						continue
					}
					if err != nil {
						if ctx.Err() == nil {
							t.Errorf("%s read partition %d: %v", consumer, partition, err)
						}
						return
					}
					acquisitionID := fmt.Sprintf("%s-%s", consumer, token.SchedulerID)
					acquired, err := store.acquireWithID(ctx, group, consumer, acquisitionID, *token)
					if err != nil {
						t.Errorf("%s acquire %s: %v", consumer, token.OrderKey, err)
						return
					}
					if acquired.Code != bstatus.SequenceQueueCodeAcquired {
						continue
					}

					var message struct {
						ID       string `json:"id"`
						OrderKey string `json:"orderKey"`
						Sequence int    `json:"sequence"`
					}
					if err := json.Unmarshal([]byte(acquired.Head), &message); err != nil {
						t.Errorf("decode acquired message: %v", err)
						return
					}

					started := time.Now()
					mutex.Lock()
					active[message.OrderKey]++
					if active[message.OrderKey] > 1 {
						overlaps[message.OrderKey]++
					}
					mutex.Unlock()
					t.Logf("CONSUME-START consumer=%s orderKey=%s sequence=%02d id=%s", consumer, message.OrderKey, message.Sequence, message.ID)

					time.Sleep(50 * time.Millisecond)
					finished := time.Now()
					mutex.Lock()
					active[message.OrderKey]--
					records = append(records, consumption{message.OrderKey, consumer, message.ID, message.Sequence, started, finished})
					mutex.Unlock()
					t.Logf("CONSUME-END   consumer=%s orderKey=%s sequence=%02d id=%s duration=%s", consumer, message.OrderKey, message.Sequence, message.ID, finished.Sub(started).Round(time.Millisecond))

					finalized, err := store.finalizeWithID(ctx, group, consumer, acquisitionID, *token, acquired.Head)
					if err != nil {
						t.Errorf("%s finalize %s: %v", consumer, message.ID, err)
						return
					}
					if finalized.Code != bstatus.SequenceQueueCodeSuccessor && finalized.Code != bstatus.SequenceQueueCodeEmpty {
						t.Errorf("%s finalize %s returned %s", consumer, message.ID, finalized.Code)
						return
					}
					completed.Add(1)
					processed = true
					break
				}
				if !processed {
					time.Sleep(10 * time.Millisecond)
				}
			}
		})
	}
	workers.Wait()
	if err := ctx.Err(); err != nil {
		t.Fatalf("test timed out after consuming %d/%d messages: %v", completed.Load(), total, err)
	}

	mutex.Lock()
	defer mutex.Unlock()
	sort.Slice(records, func(i, j int) bool { return records[i].started.Before(records[j].started) })
	lastSequence := make(map[string]int)
	for _, record := range records {
		want := lastSequence[record.orderKey] + 1
		if record.sequence != want {
			t.Errorf("order key %s consumed sequence %d after %d; want %d", record.orderKey, record.sequence, lastSequence[record.orderKey], want)
		}
		lastSequence[record.orderKey] = record.sequence
	}
	for _, orderKey := range []string{"order-a", "order-b", "order-c"} {
		t.Logf("RESULT orderKey=%s consumed=%d expected=%d overlaps=%d order_ok=%t", orderKey, lastSequence[orderKey], messageCounts[orderKey], overlaps[orderKey], lastSequence[orderKey] == messageCounts[orderKey])
		if overlaps[orderKey] != 0 {
			t.Errorf("order key %s had %d concurrent consumption overlaps", orderKey, overlaps[orderKey])
		}
		if lastSequence[orderKey] != messageCounts[orderKey] {
			t.Errorf("order key %s consumed %d messages; want %d", orderKey, lastSequence[orderKey], messageCounts[orderKey])
		}
	}
	t.Logf("SUMMARY consumers=4 published=%d consumed=%d same_order_key_overlaps=%d", total, len(records), overlaps["order-a"]+overlaps["order-b"]+overlaps["order-c"])
}
