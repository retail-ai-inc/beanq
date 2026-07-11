package bredis

import (
	"strings"
	"testing"

	"github.com/retail-ai-inc/beanq/v4/helper/tool"
)

func TestSequenceQueueTopologyNormalizesRouteAndPartitions(t *testing.T) {
	topology := newSequenceQueueTopology("prefix", "", "", 0)
	if topology.channel != "default-channel" || topology.topic != "default-topic" {
		t.Fatalf("route = %q/%q, want defaults", topology.channel, topology.topic)
	}
	if got := topology.partitions; got != 1 {
		t.Fatalf("partition count = %d, want 1", got)
	}

	configured := newSequenceQueueTopology("prefix", "channel", "topic", 17)
	if got := configured.partitions; got != 17 {
		t.Fatalf("partition count = %d, want 17", got)
	}
}

func TestSequenceQueueTopologyPartitionUsesHashKeySemantics(t *testing.T) {
	topology := newSequenceQueueTopology("prefix", "channel", "topic", 11)
	for _, customerID := range []string{"customer-1", "customer-2", "", "顾客"} {
		got := topology.partition(customerID)
		if got < 0 || got >= topology.partitions {
			t.Fatalf("partition(%q) = %d, out of range", customerID, got)
		}
		if want := int64(tool.HashKey([]byte(customerID), 11)); got != want {
			t.Fatalf("partition(%q) = %d, want HashKey result %d", customerID, got, want)
		}
		if again := topology.partition(customerID); again != got {
			t.Fatalf("partition(%q) changed from %d to %d", customerID, got, again)
		}
	}
}

func TestSequenceQueueDigestUsesUnambiguousLengthPrefixes(t *testing.T) {
	if sequenceQueueDigest("ab", "c", "d") == sequenceQueueDigest("a", "bc", "d") {
		t.Fatal("queue digest collided for different component boundaries")
	}
	if sequenceQueueCustomerDigest("customer") != sequenceQueueCustomerDigest("customer") {
		t.Fatal("customer digest is not deterministic")
	}
	if len(sequenceQueueCustomerDigest("customer")) != 64 {
		t.Fatalf("customer digest length = %d, want 64", len(sequenceQueueCustomerDigest("customer")))
	}
}

func TestSequenceQueueTopologyKeysUseInternalTagFirst(t *testing.T) {
	topology := newSequenceQueueTopology("prefix{injected}", "channel{injected}", "topic{injected}", 4)
	customerID := "customer{injected}"
	partition := int64(2)
	keys := []string{
		topology.schedulerKey(partition),
		topology.pendingKey(partition),
		topology.customerListKey(partition, customerID),
		topology.customerStateKey(partition, customerID),
	}

	tag := topology.partitionTag(partition)
	for _, key := range keys {
		if !strings.HasPrefix(key, tag+":") {
			t.Fatalf("key %q does not start with internal tag %q", key, tag)
		}
		if firstRedisHashTag(key) != strings.Trim(tag, "{}") {
			t.Fatalf("key %q selected injected hash tag", key)
		}
	}
	if firstRedisHashTag(topology.metadataKey()) != strings.Trim(topology.metadataTag(), "{}") {
		t.Fatalf("metadata key %q selected injected hash tag", topology.metadataKey())
	}
}

func TestSequenceQueueTopologyKeysShareOnlyTheirPartitionSlot(t *testing.T) {
	topology := newSequenceQueueTopology("prefix", "channel", "topic", 4)
	partition0 := []string{
		topology.schedulerKey(0),
		topology.pendingKey(0),
		topology.customerListKey(0, "customer-a"),
		topology.customerStateKey(0, "customer-b"),
	}
	wantTag := firstRedisHashTag(partition0[0])
	for _, key := range partition0[1:] {
		if got := firstRedisHashTag(key); got != wantTag {
			t.Fatalf("same-partition key %q tag = %q, want %q", key, got, wantTag)
		}
	}
	if got := firstRedisHashTag(topology.schedulerKey(1)); got == wantTag {
		t.Fatalf("different partitions reused tag %q", got)
	}
}

func firstRedisHashTag(key string) string {
	open := strings.IndexByte(key, '{')
	if open < 0 {
		return ""
	}
	close := strings.IndexByte(key[open+1:], '}')
	if close <= 0 {
		return ""
	}
	return key[open+1 : open+1+close]
}
