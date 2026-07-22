package bredis

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestSequenceQueuePartitionKeysUseOneHashSlot(t *testing.T) {
	topology := newSequenceQueueTopology("prefix", "channel", "topic", 8)
	partition := int64(3)
	keys := []string{
		topology.schedulerKey(partition),
		topology.partitionStateKey(partition),
		topology.isolationKey(partition),
		topology.orderListKey(partition, "order-a"),
		topology.orderStateKey(partition, "order-a"),
	}

	tag := topology.partitionTag(partition)
	for _, key := range keys {
		if !strings.Contains(key, tag) {
			t.Fatalf("key %q does not contain partition hash tag %q", key, tag)
		}
		if !strings.Contains(key, ":sequence_queue:") {
			t.Fatalf("key %q is not in v3 namespace", key)
		}
		if strings.Contains(key, ":v2:") {
			t.Fatalf("key %q repeats the version outside its hash tag", key)
		}
	}
}

func TestSequenceQueueDifferentOrderKeysHaveDifferentStateKeys(t *testing.T) {
	topology := newSequenceQueueTopology("prefix", "channel", "topic", 1)
	if topology.orderStateKey(0, "order-a") == topology.orderStateKey(0, "order-b") {
		t.Fatal("different order keys share a state key")
	}
}

func TestSequenceQueueScriptCatalogContainsQueueScripts(t *testing.T) {
	catalog := DefaultScriptCatalog()
	if catalog != DefaultScriptCatalog() {
		t.Fatal("default script catalog is not shared")
	}
	for _, name := range []string{ScriptSequenceQueueEnqueue, ScriptSequenceQueueLease, ScriptSequenceQueueFinalize} {
		if catalog.scripts[name] == nil {
			t.Fatalf("script %q is not registered", name)
		}
	}

	first := newSequenceQueueStore(nil, newSequenceQueueTopology("p", "c", "t", 1), 10, 0)
	second := newSequenceQueueStore(nil, newSequenceQueueTopology("p", "c", "t", 1), 10, 0)
	if first.scripts != second.scripts {
		t.Fatal("sequence queue stores do not share the default script catalog")
	}
}

func TestDefaultScriptCatalogSupportsConcurrentReads(t *testing.T) {
	const readers = 32
	var wait sync.WaitGroup
	for range readers {
		wait.Go(func() {
			catalog := DefaultScriptCatalog()
			for _, name := range []string{ScriptSequenceQueueEnqueue, ScriptSequenceQueueLease, ScriptSequenceQueueFinalize} {
				if catalog.scripts[name] == nil {
					t.Errorf("script %q is not registered", name)
				}
			}
		})
	}
	wait.Wait()
}

func TestNewScriptCatalogCopiesInput(t *testing.T) {
	scripts := map[string]*redis.Script{"original": SequenceQueueEnqueueScript}
	catalog := NewScriptCatalog(scripts)
	scripts["original"] = SequenceQueueLeaseScript
	scripts["added"] = SequenceQueueFinalizeScript

	if catalog.scripts["original"] != SequenceQueueEnqueueScript {
		t.Fatal("catalog changed when caller mutated the input map")
	}
	if catalog.scripts["added"] != nil {
		t.Fatal("catalog contains an entry added after construction")
	}
}

func TestSequenceQueueCanonicalConfigIncludesCapacity(t *testing.T) {
	store := newSequenceQueueStore(nil, newSequenceQueueTopology("p", "c", "t", 7), 123, 0)
	if got, want := store.canonicalConfig(), "schema=3;partitions=7;capacity=123"; got != want {
		t.Fatalf("canonical config = %q, want %q", got, want)
	}
}

func TestSequenceQueueMetadataReadsExistingConfigWithoutWrite(t *testing.T) {
	client := &sequenceQueueMetadataClient{}
	store := newSequenceQueueStore(client, newSequenceQueueTopology("p", "c", "t", 7), 123, 0)
	client.hget = []sequenceQueueMetadataResult{{value: store.canonicalConfig()}}

	if err := store.ensureMetadata(context.Background()); err != nil {
		t.Fatalf("ensureMetadata: %v", err)
	}
	if got, want := strings.Join(client.calls, ","), "HGET"; got != want {
		t.Fatalf("metadata calls = %q, want %q", got, want)
	}
}

func TestSequenceQueueMetadataInitializesAfterMissingRead(t *testing.T) {
	client := &sequenceQueueMetadataClient{}
	store := newSequenceQueueStore(client, newSequenceQueueTopology("p", "c", "t", 7), 123, 0)
	client.hget = []sequenceQueueMetadataResult{{err: redis.Nil}, {value: store.canonicalConfig()}}

	if err := store.ensureMetadata(context.Background()); err != nil {
		t.Fatalf("ensureMetadata: %v", err)
	}
	if got, want := strings.Join(client.calls, ","), "HGET,HSETNX,HGET"; got != want {
		t.Fatalf("metadata calls = %q, want %q", got, want)
	}
}

func TestSequenceQueueMetadataRejectsExistingMismatchWithoutWrite(t *testing.T) {
	client := &sequenceQueueMetadataClient{hget: []sequenceQueueMetadataResult{{value: "other"}}}
	store := newSequenceQueueStore(client, newSequenceQueueTopology("p", "c", "t", 7), 123, 0)

	err := store.ensureMetadata(context.Background())
	if err == nil || !strings.Contains(err.Error(), "metadata mismatch") {
		t.Fatalf("ensureMetadata error = %v, want metadata mismatch", err)
	}
	if got, want := strings.Join(client.calls, ","), "HGET"; got != want {
		t.Fatalf("metadata calls = %q, want %q", got, want)
	}
}

func TestSequenceQueueMetadataPreservesOperationErrors(t *testing.T) {
	tests := []struct {
		name       string
		client     *sequenceQueueMetadataClient
		wantPrefix string
	}{
		{name: "initial read", client: &sequenceQueueMetadataClient{hget: []sequenceQueueMetadataResult{{err: context.Canceled}}}, wantPrefix: "read sequence queue metadata"},
		{name: "initialize", client: &sequenceQueueMetadataClient{hget: []sequenceQueueMetadataResult{{err: redis.Nil}}, hsetErr: context.Canceled}, wantPrefix: "initialize sequence queue metadata"},
		{name: "final read", client: &sequenceQueueMetadataClient{hget: []sequenceQueueMetadataResult{{err: redis.Nil}, {err: context.Canceled}}}, wantPrefix: "read sequence queue metadata"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newSequenceQueueStore(test.client, newSequenceQueueTopology("p", "c", "t", 7), 123, 0)
			err := store.ensureMetadata(context.Background())
			if err == nil || !strings.Contains(err.Error(), test.wantPrefix) {
				t.Fatalf("ensureMetadata error = %v, want prefix %q", err, test.wantPrefix)
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("ensureMetadata error = %v, want context cancellation", err)
			}
		})
	}
}

type sequenceQueueMetadataResult struct {
	value string
	err   error
}

type sequenceQueueMetadataClient struct {
	redis.UniversalClient
	hget    []sequenceQueueMetadataResult
	hsetErr error
	calls   []string
}

func (c *sequenceQueueMetadataClient) HGet(context.Context, string, string) *redis.StringCmd {
	c.calls = append(c.calls, "HGET")
	result := c.hget[0]
	c.hget = c.hget[1:]
	return redis.NewStringResult(result.value, result.err)
}

func (c *sequenceQueueMetadataClient) HSetNX(context.Context, string, string, any) *redis.BoolCmd {
	c.calls = append(c.calls, "HSETNX")
	return redis.NewBoolResult(c.hsetErr == nil, c.hsetErr)
}

func TestSequenceQueueStoreRejectsTokenFromWrongPartition(t *testing.T) {
	topology := newSequenceQueueTopology("p", "c", "t", 7)
	store := newSequenceQueueStore(nil, topology, 10, 0)
	orderKey := "order"
	wrong := (topology.partition(orderKey) + 1) % topology.partitions
	if err := store.validateToken(sequenceQueueToken{Partition: wrong, SchedulerID: "1-0", OrderKey: orderKey}); err == nil {
		t.Fatal("validateToken accepted mismatched order-key partition")
	}
}

func TestTokenFromMessageRequiresCanonicalFields(t *testing.T) {
	valid, err := tokenFromMessage(2, redis.XMessage{ID: "1-0", Values: map[string]any{"order_key": "c"}})
	if err != nil || valid.OrderKey != "c" || valid.SchedulerID != "1-0" || valid.Partition != 2 {
		t.Fatalf("valid token = %#v, %v", valid, err)
	}
	for _, message := range []redis.XMessage{
		{ID: "1-0", Values: map[string]any{}},
		{ID: "1-0", Values: map[string]any{"order_key": ""}},
		{ID: "1-0", Values: map[string]any{"order_key": "c", "extra": "x"}},
	} {
		if _, err := tokenFromMessage(0, message); err == nil {
			t.Fatalf("tokenFromMessage(%#v) accepted non-canonical fields", message.Values)
		}
	}
}
