//go:build integration || ci
// +build integration ci

package bredis

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	public "github.com/retail-ai-inc/beanq/v4/internal"
)

type normalIntegrationProcessLogger struct{}

func (normalIntegrationProcessLogger) AddLog(context.Context, map[string]any) error { return nil }

func TestNormalQueueFivePublishersTenConsumers(t *testing.T) {
	const (
		publisherCount       = 5
		consumerCount        = 10
		messagesPerPublisher = 200
		partitions           = 16
		totalMessages        = publisherCount * messagesPerPublisher
	)

	testCtx, testCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer testCancel()
	client, configuredPrefix := newSequenceQueueIntegrationClient(t, testCtx)
	t.Cleanup(func() { _ = client.Close() })

	prefix := fmt.Sprintf("%snormal_multi_%d", configuredPrefix, time.Now().UnixNano())
	channel, topic := "normal-integration", "five-publishers-ten-consumers"
	topology := newNormalQueueTopology(prefix, channel, topic, partitions)
	t.Cleanup(func() { cleanupSequenceQueueIntegrationKeys(t, context.Background(), client, prefix) })

	consumerCtx, stopConsumers := context.WithCancel(testCtx)
	var consumerWait sync.WaitGroup

	consumed := make(map[string]int, totalMessages)
	var consumedMutex sync.Mutex
	completed := make(chan struct{})
	var completeOnce sync.Once
	handler := public.NewCallbackWithRetry(func(_ context.Context, data map[string]any, _ ...int) (int, error) {
		messageID := fmt.Sprint(data["id"])
		consumedMutex.Lock()
		consumed[messageID]++
		if len(consumed) == totalMessages {
			completeOnce.Do(func() { close(completed) })
		}
		consumedMutex.Unlock()
		return 0, nil
	}, func(context.Context, error) {})

	for consumerNumber := 0; consumerNumber < consumerCount; consumerNumber++ {
		base := newQueueBase(queueBaseOptions{client: client, prefix: prefix, deadLetterIdle: time.Minute, consumerPoolSize: 1})
		base.processLogger = normalIntegrationProcessLogger{}
		runtime := newNormalQueueRuntime(newNormalQueueStore(client, topology, 2000), &base)
		consumerWait.Go(func() { runtime.run(consumerCtx, channel, topic, handler) })
	}

	started := time.Now()
	publishErrors := make(chan error, publisherCount)
	var publisherWait sync.WaitGroup
	partitionCounts := make([]int, partitions)
	var partitionMutex sync.Mutex
	for publisherNumber := 0; publisherNumber < publisherCount; publisherNumber++ {
		publisherNumber := publisherNumber
		publisherWait.Go(func() {
			publisher := newNormalWithPartitions(client, prefix, 2000, partitions, 1, time.Minute, nil)
			for sequence := 0; sequence < messagesPerPublisher; sequence++ {
				messageID := fmt.Sprintf("publisher-%02d-message-%04d", publisherNumber, sequence)
				data := map[string]any{
					"id": messageID, "channel": channel, "topic": topic, "payload": messageID,
					"retry": 0, "timeToRun": 5 * time.Second,
				}
				if err := publisher.Publish(testCtx, data); err != nil {
					publishErrors <- fmt.Errorf("publish %s: %w", messageID, err)
					return
				}
				partitionMutex.Lock()
				partitionCounts[topology.partition(messageID)]++
				partitionMutex.Unlock()
			}
		})
	}
	publisherWait.Wait()
	close(publishErrors)
	for err := range publishErrors {
		if err != nil {
			t.Fatal(err)
		}
	}

	select {
	case <-completed:
	case <-testCtx.Done():
		consumedMutex.Lock()
		consumedCount := len(consumed)
		consumedMutex.Unlock()
		var remaining, pending int64
		for partition := int64(0); partition < partitions; partition++ {
			remaining += client.XLen(context.Background(), topology.streamKey(partition)).Val()
			if summary, err := client.XPending(context.Background(), topology.streamKey(partition), channel).Result(); err == nil {
				pending += summary.Count
			}
		}
		t.Fatalf("timed out consuming messages: unique=%d want=%d remaining=%d pending=%d: %v", consumedCount, totalMessages, remaining, pending, testCtx.Err())
	}
	stopConsumers()
	consumerWait.Wait()
	elapsed := time.Since(started)

	consumedMutex.Lock()
	defer consumedMutex.Unlock()
	duplicates := 0
	for messageID, count := range consumed {
		if count != 1 {
			duplicates += count - 1
			t.Errorf("message %s consumed %d times", messageID, count)
		}
	}
	if len(consumed) != totalMessages {
		t.Fatalf("unique consumed=%d, want=%d", len(consumed), totalMessages)
	}

	usedPartitions := 0
	remaining := int64(0)
	for partition := int64(0); partition < partitions; partition++ {
		if partitionCounts[partition] > 0 {
			usedPartitions++
		}
		length, err := client.XLen(context.Background(), topology.streamKey(partition)).Result()
		if err != nil {
			t.Fatalf("read partition %d length: %v", partition, err)
		}
		remaining += length
	}
	if usedPartitions < partitions/2 {
		t.Errorf("messages used only %d/%d partitions", usedPartitions, partitions)
	}
	if remaining != 0 {
		t.Errorf("remaining stream messages=%d, want=0", remaining)
	}
	t.Logf("SUMMARY publishers=%d consumers=%d partitions=%d used_partitions=%d published=%d unique_consumed=%d duplicates=%d remaining=%d elapsed=%s rate=%.0f msg/s",
		publisherCount, consumerCount, partitions, usedPartitions, totalMessages, len(consumed), duplicates, remaining, elapsed.Round(time.Millisecond), float64(totalMessages)/elapsed.Seconds())
}
