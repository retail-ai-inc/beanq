//go:build integration || ci
// +build integration ci

package bredis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
)

func TestNormalAndDelayStatusRedisIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, configuredPrefix := newSequenceQueueIntegrationClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	prefix := fmt.Sprintf("%sstatus_it_%d", configuredPrefix, time.Now().UnixNano())
	t.Cleanup(func() {
		cleanupRedisKeysWithPrefix(t, context.Background(), client, prefix)
	})
	const partitions = int64(7)
	processLog := NewProcessLogWithPartitions(client, prefix, partitions, 11)
	status := NewStatusWithPartitions(client, prefix, partitions, 11)

	for _, tt := range []struct {
		name     string
		moodType btype.MoodType
	}{
		{name: "normal", moodType: btype.NORMAL},
		{name: "delay", moodType: btype.DELAY},
	} {
		t.Run(tt.name, func(t *testing.T) {
			channel := tt.name + "-channel"
			topic := "status-topic"
			id := tt.name + "-message-01"
			data := map[string]any{
				"channel": channel, "topic": topic, "id": id,
				"moodType": tt.moodType, "status": bstatus.StatusSuccess,
			}
			if err := processLog.AddLog(ctx, data); err != nil {
				t.Fatalf("AddLog: %v", err)
			}
			got, err := status.Status(ctx, channel, topic, id)
			if err != nil {
				t.Fatalf("Status: %v", err)
			}
			if got["id"] != id || got["status"] != bstatus.StatusSuccess {
				t.Fatalf("status = %#v, want id %q and success", got, id)
			}
		})
	}
}

type sequenceQueueIntegrationConfig struct {
	Redis struct {
		IsCluster    bool   `json:"isCluster"`
		Host         string `json:"host"`
		Port         string `json:"port"`
		Username     string `json:"username"`
		Password     string `json:"password"`
		Database     int    `json:"database"`
		Prefix       string `json:"prefix"`
		MaxRetries   int    `json:"maxRetries"`
		PoolSize     int    `json:"poolSize"`
		MinIdleConns int    `json:"minIdleConnections"`
		DialTimeout  string `json:"dialTimeout"`
		ReadTimeout  string `json:"readTimeout"`
		WriteTimeout string `json:"writeTimeout"`
		PoolTimeout  string `json:"poolTimeout"`
		SSL          struct {
			On                bool   `json:"on"`
			CertFile          string `json:"certFile"`
			VerifyCertificate bool   `json:"verifyCertificate"`
		} `json:"ssl"`
	} `json:"redis"`
}

func TestSequenceQueueStoreRedisIntegration(t *testing.T) {
	ctx := context.Background()
	client, prefix := newSequenceQueueIntegrationClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	prefix = fmt.Sprintf("%ssq_it_%d", prefix, time.Now().UnixNano())
	topology := newSequenceQueueTopology(prefix, "integration", "sequence-queue", 1)
	store := newSequenceQueueStore(client, topology, 2, 5*time.Second)
	cleanupSequenceQueueIntegrationKeys(t, ctx, client, prefix)
	t.Cleanup(func() { cleanupSequenceQueueIntegrationKeys(t, context.Background(), client, prefix) })

	t.Run("metadata", func(t *testing.T) {
		if err := store.ensureMetadata(ctx); err != nil {
			t.Fatalf("ensureMetadata: %v", err)
		}
		got, err := client.HGet(ctx, topology.metadataKey(), "canonical_config").Result()
		if err != nil {
			t.Fatalf("read metadata: %v", err)
		}
		if want := store.canonicalConfig(); got != want {
			t.Fatalf("canonical_config = %q, want %q", got, want)
		}

		if err := store.ensureMetadata(ctx); err != nil {
			t.Fatalf("repeat ensureMetadata: %v", err)
		}

		incompatible := newSequenceQueueStore(client, topology, 3, time.Second)
		if err := incompatible.ensureMetadata(ctx); err == nil || !strings.Contains(err.Error(), "metadata mismatch") {
			t.Fatalf("incompatible ensureMetadata error = %v, want metadata mismatch", err)
		}

		if err := client.Del(ctx, topology.metadataKey()).Err(); err != nil {
			t.Fatalf("delete metadata: %v", err)
		}
		if err := incompatible.ensureMetadata(ctx); err != nil {
			t.Fatalf("reinitialize incompatible metadata: %v", err)
		}
		if err := store.ensureMetadata(ctx); err == nil || !strings.Contains(err.Error(), "metadata mismatch") {
			t.Fatalf("original store error after incompatible rebuild = %v, want metadata mismatch", err)
		}
	})

	t.Run("concurrent metadata initialization", func(t *testing.T) {
		concurrentPrefix := prefix + "_concurrent"
		concurrentTopology := newSequenceQueueTopology(concurrentPrefix, "integration", "sequence-queue", 1)
		concurrentStore := newSequenceQueueStore(client, concurrentTopology, 2, time.Second)
		t.Cleanup(func() { cleanupSequenceQueueIntegrationKeys(t, context.Background(), client, concurrentPrefix) })

		const publishers = 16
		start := make(chan struct{})
		errs := make(chan error, publishers)
		var wait sync.WaitGroup
		for range publishers {
			wait.Go(func() {
				<-start
				errs <- concurrentStore.ensureMetadata(ctx)
			})
		}
		close(start)
		wait.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatalf("concurrent ensureMetadata: %v", err)
			}
		}
		got, err := client.HGet(ctx, concurrentTopology.metadataKey(), "canonical_config").Result()
		if err != nil {
			t.Fatalf("read concurrent metadata: %v", err)
		}
		if want := concurrentStore.canonicalConfig(); got != want {
			t.Fatalf("concurrent canonical_config = %q, want %q", got, want)
		}
	})

	t.Run("enqueue scheduled queued full and hash pending", func(t *testing.T) {
		customerID := "enqueue-customer"
		first, err := store.enqueue(ctx, customerID, map[string]any{"id": "first"})
		if err != nil {
			t.Fatalf("first enqueue: %v", err)
		}
		if first.Code != bstatus.SequenceQueueCodeScheduled || first.SchedulerID == "" || first.Pending != 1 {
			t.Fatalf("first enqueue = %#v, want SCHEDULED with pending 1", first)
		}

		second, err := store.enqueue(ctx, customerID, map[string]any{"id": "second"})
		if err != nil {
			t.Fatalf("second enqueue: %v", err)
		}
		if second.Code != bstatus.SequenceQueueCodeQueued || second.SchedulerID != first.SchedulerID || second.Pending != 2 {
			t.Fatalf("second enqueue = %#v, want QUEUED on scheduler %q with pending 2", second, first.SchedulerID)
		}

		full, err := store.enqueue(ctx, customerID, map[string]any{"id": "third"})
		if err != nil {
			t.Fatalf("full enqueue: %v", err)
		}
		if full.Code != bstatus.SequenceQueueCodeFull || full.Pending != 2 {
			t.Fatalf("full enqueue = %#v, want FULL with pending 2", full)
		}

		pending, err := client.HGet(ctx, topology.partitionStateKey(0), "count").Int64()
		if err != nil || pending != 2 {
			t.Fatalf("partition hash pending = %d, %v; want 2", pending, err)
		}
		state, err := client.HGetAll(ctx, topology.orderStateKey(0, customerID)).Result()
		if err != nil {
			t.Fatalf("read customer state: %v", err)
		}
		if state["order_key"] != customerID || state["scheduler_id"] != first.SchedulerID {
			t.Fatalf("customer state = %#v", state)
		}
	})
}

func TestSequenceQueueLeaseAndFinalizeRedisIntegration(t *testing.T) {
	ctx := context.Background()
	client, prefix := newSequenceQueueIntegrationClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	prefix = fmt.Sprintf("%ssq_it_%d", prefix, time.Now().UnixNano())
	topology := newSequenceQueueTopology(prefix, "integration", "lease-finalize", 1)
	store := newSequenceQueueStore(client, topology, 10, 5*time.Second)
	cleanupSequenceQueueIntegrationKeys(t, ctx, client, prefix)
	t.Cleanup(func() { cleanupSequenceQueueIntegrationKeys(t, context.Background(), client, prefix) })

	const (
		group      = "workers"
		consumer   = "consumer-a"
		customerID = "lease-customer"
	)
	if err := store.bootstrapGroups(ctx, group); err != nil {
		t.Fatalf("bootstrapGroups: %v", err)
	}
	if _, err := store.enqueue(ctx, customerID, map[string]any{"id": "first"}); err != nil {
		t.Fatalf("enqueue first: %v", err)
	}
	if _, err := store.enqueue(ctx, customerID, map[string]any{"id": "second"}); err != nil {
		t.Fatalf("enqueue second: %v", err)
	}

	token, err := store.readOne(ctx, group, consumer, 0, time.Second)
	if err != nil {
		t.Fatalf("read first scheduler token: %v", err)
	}
	firstHead, err := client.LIndex(ctx, topology.orderListKey(0, customerID), 0).Result()
	if err != nil {
		t.Fatalf("read first head: %v", err)
	}
	acquired, err := store.acquireWithID(ctx, group, consumer, "acquisition-1", *token)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if acquired.Code != bstatus.SequenceQueueCodeAcquired || acquired.Head != firstHead || acquired.DeadlineMS <= time.Now().UnixMilli() {
		t.Fatalf("acquire result = %#v", acquired)
	}
	if phase, err := client.HGet(ctx, topology.orderStateKey(0, customerID), "phase").Result(); err != nil || phase != "owned" {
		t.Fatalf("phase after acquire = %q, %v; want owned", phase, err)
	}

	renewed, err := store.renew(ctx, group, consumer, "acquisition-1", *token)
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if renewed.Code != bstatus.SequenceQueueCodeRenewed || renewed.DeadlineMS < acquired.DeadlineMS {
		t.Fatalf("renew result = %#v, acquired deadline %d", renewed, acquired.DeadlineMS)
	}

	stale, err := store.finalizeWithID(ctx, group, consumer, "stale-acquisition", *token, firstHead)
	if err != nil {
		t.Fatalf("stale finalize: %v", err)
	}
	if stale.Code != bstatus.SequenceQueueCodeStaleAcquisition {
		t.Fatalf("stale finalize = %#v, want STALE_ACQUISITION", stale)
	}

	successor, err := store.finalizeWithID(ctx, group, consumer, "acquisition-1", *token, firstHead)
	if err != nil {
		t.Fatalf("finalize successor: %v", err)
	}
	if successor.Code != bstatus.SequenceQueueCodeSuccessor || successor.SchedulerID == "" || successor.Remaining != 1 {
		t.Fatalf("successor finalize = %#v", successor)
	}

	successorToken, err := store.readOne(ctx, group, consumer, 0, time.Second)
	if err != nil {
		t.Fatalf("read successor token: %v", err)
	}
	if successorToken.SchedulerID != successor.SchedulerID {
		t.Fatalf("successor token ID = %q, want %q", successorToken.SchedulerID, successor.SchedulerID)
	}
	secondHead, err := client.LIndex(ctx, topology.orderListKey(0, customerID), 0).Result()
	if err != nil {
		t.Fatalf("read second head: %v", err)
	}
	if _, err := store.acquireWithID(ctx, group, consumer, "acquisition-2", *successorToken); err != nil {
		t.Fatalf("acquire successor: %v", err)
	}
	empty, err := store.finalizeWithID(ctx, group, consumer, "acquisition-2", *successorToken, secondHead)
	if err != nil {
		t.Fatalf("finalize empty: %v", err)
	}
	if empty.Code != bstatus.SequenceQueueCodeEmpty || empty.Remaining != 0 {
		t.Fatalf("empty finalize = %#v", empty)
	}
	if exists, err := client.Exists(ctx, topology.orderListKey(0, customerID), topology.orderStateKey(0, customerID)).Result(); err != nil || exists != 0 {
		t.Fatalf("customer keys exist = %d, %v; want 0", exists, err)
	}
	if pending, err := client.HGet(ctx, topology.partitionStateKey(0), "count").Int64(); err != nil || pending != 0 {
		t.Fatalf("count after empty = %d, %v; want 0", pending, err)
	}
}

func TestSequenceQueueAbnormalEmptyCleanupRedisIntegration(t *testing.T) {
	ctx := context.Background()
	client, prefix := newSequenceQueueIntegrationClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	prefix = fmt.Sprintf("%ssq_it_%d", prefix, time.Now().UnixNano())
	topology := newSequenceQueueTopology(prefix, "integration", "empty-cleanup", 1)
	store := newSequenceQueueStore(client, topology, 10, 5*time.Second)
	t.Cleanup(func() { cleanupSequenceQueueIntegrationKeys(t, context.Background(), client, prefix) })

	const (
		group      = "workers"
		consumer   = "consumer-a"
		customerID = "empty-customer"
	)
	if err := store.bootstrapGroups(ctx, group); err != nil {
		t.Fatalf("bootstrapGroups: %v", err)
	}
	if _, err := store.enqueue(ctx, customerID, map[string]any{"id": "lost"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	token, err := store.readOne(ctx, group, consumer, 0, time.Second)
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	acquired, err := store.acquireWithID(ctx, group, consumer, "acquisition-empty", *token)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := client.Del(ctx, topology.orderListKey(0, customerID)).Err(); err != nil {
		t.Fatalf("delete customer list: %v", err)
	}
	result, err := store.finalizeWithID(ctx, group, consumer, "acquisition-empty", *token, acquired.Head)
	if err != nil {
		t.Fatalf("finalize abnormal empty: %v", err)
	}
	if result.Code != bstatus.SequenceQueueCodeEmptyCleaned {
		t.Fatalf("abnormal empty finalize = %#v, want EMPTY_CLEANED", result)
	}
	if exists, err := client.Exists(ctx, topology.orderStateKey(0, customerID)).Result(); err != nil || exists != 0 {
		t.Fatalf("customer state exists = %d, %v; want 0", exists, err)
	}
	entries, err := client.XRange(ctx, topology.isolationKey(0), "-", "+").Result()
	if err != nil {
		t.Fatalf("read isolation stream: %v", err)
	}
	if len(entries) != 1 || entries[0].Values["reason"] != "empty_queue" {
		t.Fatalf("isolation entries = %#v, want one empty_queue entry", entries)
	}
}

func TestSequenceQueueAutoClaimCursorRedisIntegration(t *testing.T) {
	ctx := context.Background()
	client, prefix := newSequenceQueueIntegrationClient(t, ctx)
	t.Cleanup(func() { _ = client.Close() })

	prefix = fmt.Sprintf("%ssq_it_%d", prefix, time.Now().UnixNano())
	topology := newSequenceQueueTopology(prefix, "integration", "autoclaim", 1)
	store := newSequenceQueueStore(client, topology, 10, 5*time.Second)
	t.Cleanup(func() { cleanupSequenceQueueIntegrationKeys(t, context.Background(), client, prefix) })

	const group = "workers"
	if err := store.bootstrapGroups(ctx, group); err != nil {
		t.Fatalf("bootstrapGroups: %v", err)
	}
	for _, customerID := range []string{"claim-a", "claim-b"} {
		if _, err := store.enqueue(ctx, customerID, map[string]any{"customer": customerID}); err != nil {
			t.Fatalf("enqueue %s: %v", customerID, err)
		}
		if _, err := store.readOne(ctx, group, "old-consumer", 0, time.Second); err != nil {
			t.Fatalf("read %s into PEL: %v", customerID, err)
		}
	}

	first, err := store.autoClaim(ctx, group, "new-consumer", 0, 0, "0-0", 1)
	if err != nil {
		t.Fatalf("first autoClaim: %v", err)
	}
	if len(first.Tokens) != 1 || first.Cursor == "" {
		t.Fatalf("first autoClaim = %#v, want one token and cursor", first)
	}
	second, err := store.autoClaim(ctx, group, "new-consumer", 0, 0, first.Cursor, 1)
	if err != nil {
		t.Fatalf("second autoClaim: %v", err)
	}
	if len(second.Tokens) != 1 || second.Cursor == "" {
		t.Fatalf("second autoClaim = %#v, want one token and cursor", second)
	}
	if first.Tokens[0].SchedulerID == second.Tokens[0].SchedulerID {
		t.Fatalf("autoClaim returned duplicate scheduler token %q", first.Tokens[0].SchedulerID)
	}
}

func newSequenceQueueIntegrationClient(t *testing.T, ctx context.Context) (redis.UniversalClient, string) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate integration test source")
	}
	configPath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "env.testing.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	var config sequenceQueueIntegrationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("decode %s: %v", configPath, err)
	}
	if config.Redis.IsCluster {
		t.Fatal("sequence queue integration test currently requires env.testing.json redis.isCluster=false")
	}

	parseDuration := func(name, value string) time.Duration {
		t.Helper()
		duration, err := time.ParseDuration(value)
		if err != nil {
			t.Fatalf("parse redis.%s %q: %v", name, value, err)
		}
		return duration
	}
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        []string{config.Redis.Host + ":" + config.Redis.Port},
		Username:     config.Redis.Username,
		Password:     config.Redis.Password,
		DB:           config.Redis.Database,
		MaxRetries:   config.Redis.MaxRetries,
		PoolSize:     config.Redis.PoolSize,
		MinIdleConns: config.Redis.MinIdleConns,
		DialTimeout:  parseDuration("dialTimeout", config.Redis.DialTimeout),
		ReadTimeout:  parseDuration("readTimeout", config.Redis.ReadTimeout),
		WriteTimeout: parseDuration("writeTimeout", config.Redis.WriteTimeout),
		PoolTimeout:  parseDuration("poolTimeout", config.Redis.PoolTimeout),
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("ping Redis configured by %s: %v", configPath, err)
	}
	return client, config.Redis.Prefix
}

func cleanupSequenceQueueIntegrationKeys(t *testing.T, ctx context.Context, client redis.UniversalClient, prefix string) {
	t.Helper()
	var cursor uint64
	for {
		keys, next, err := client.Scan(ctx, cursor, "*:"+prefix+":*", 100).Result()
		if err != nil {
			t.Errorf("scan integration keys: %v", err)
			return
		}
		if len(keys) > 0 {
			if err := client.Del(ctx, keys...).Err(); err != nil {
				t.Errorf("delete integration keys: %v", err)
				return
			}
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}

func cleanupRedisKeysWithPrefix(t *testing.T, ctx context.Context, client redis.UniversalClient, prefix string) {
	t.Helper()
	var cursor uint64
	for {
		keys, next, err := client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			t.Errorf("scan integration keys: %v", err)
			return
		}
		if len(keys) > 0 {
			if err := client.Del(ctx, keys...).Err(); err != nil {
				t.Errorf("delete integration keys: %v", err)
				return
			}
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}
