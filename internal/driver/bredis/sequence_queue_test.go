package bredis

import (
	"strings"
	"testing"
)

func TestSequenceQueuePartitionKeysUseOneHashSlot(t *testing.T) {
	topology := newSequenceQueueTopology("prefix", "channel", "topic", 8)
	partition := int64(3)
	keys := []string{
		topology.schedulerKey(partition),
		topology.pendingKey(partition),
		topology.customerListKey(partition, "customer-a"),
		topology.customerStateKey(partition, "customer-a"),
	}

	tag := topology.partitionTag(partition)
	for _, key := range keys {
		if !strings.Contains(key, tag) {
			t.Fatalf("key %q does not contain partition hash tag %q", key, tag)
		}
	}
}

func TestSequenceQueueDifferentCustomersHaveDifferentStateKeys(t *testing.T) {
	topology := newSequenceQueueTopology("prefix", "channel", "topic", 1)
	a := topology.customerStateKey(0, "customer-a")
	b := topology.customerStateKey(0, "customer-b")
	if a == b {
		t.Fatal("different customers share a state key")
	}
}

func TestSequenceQueueScriptCatalog(t *testing.T) {
	catalog := DefaultScriptCatalog()
	for _, name := range []string{
		ScriptSequenceQueueMeta,
		ScriptSequenceQueueEnqueue,
		ScriptSequenceQueueAcquire,
		ScriptSequenceQueueHeartbeat,
		ScriptSequenceQueueFinalize,
	} {
		if catalog.scripts[name] == nil {
			t.Fatalf("script %q is not registered", name)
		}
	}
}
