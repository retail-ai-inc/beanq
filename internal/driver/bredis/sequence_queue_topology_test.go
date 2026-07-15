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
	for _, orderKey := range []string{"order-1", "order-2", "", "订单"} {
		got := topology.partition(orderKey)
		if got < 0 || got >= topology.partitions {
			t.Fatalf("partition(%q) = %d, out of range", orderKey, got)
		}
		if want := int64(tool.HashKey([]byte(orderKey), 11)); got != want {
			t.Fatalf("partition(%q) = %d, want HashKey result %d", orderKey, got, want)
		}
		if again := topology.partition(orderKey); again != got {
			t.Fatalf("partition(%q) changed from %d to %d", orderKey, got, again)
		}
	}
}

func TestSequenceQueueDigestUsesUnambiguousLengthPrefixes(t *testing.T) {
	if sequenceQueueDigest("ab", "c", "d") == sequenceQueueDigest("a", "bc", "d") {
		t.Fatal("queue digest collided for different component boundaries")
	}

	if len(sequenceQueueOrderKeyDigest("order")) != 64 {
		t.Fatalf("order-key digest length = %d, want 64", len(sequenceQueueOrderKeyDigest("order")))
	}
}

func TestSequenceQueueTopologyKeysUseRouteThenInternalTag(t *testing.T) {
	topology := newSequenceQueueTopology("prefix", "channel", "topic", 4)
	orderKey := "order{injected}"
	partition := int64(2)
	keys := []string{
		topology.schedulerKey(partition),
		topology.partitionStateKey(partition),
		topology.isolationKey(partition),
		topology.orderListKey(partition, orderKey),
		topology.orderStateKey(partition, orderKey),
	}

	tag := topology.partitionTag(partition)
	for _, key := range keys {
		if !strings.HasPrefix(key, "prefix:channel:topic:"+tag+":") {
			t.Fatalf("key %q does not start with configured route and internal tag %q", key, tag)
		}
		if firstRedisHashTag(key) != strings.Trim(tag, "{}") {
			t.Fatalf("key %q selected injected hash tag", key)
		}
	}
	if !strings.HasPrefix(topology.metadataKey(), "prefix:channel:topic:"+topology.metadataTag()+":") {
		t.Fatalf("metadata key %q does not start with configured route", topology.metadataKey())
	}
	if firstRedisHashTag(topology.metadataKey()) != strings.Trim(topology.metadataTag(), "{}") {
		t.Fatalf("metadata key %q selected injected hash tag", topology.metadataKey())
	}
	if !strings.Contains(topology.metadataKey(), ":sequence_queue:") {
		t.Fatalf("metadata key %q is not in v3 namespace", topology.metadataKey())
	}
	if strings.Contains(topology.metadataKey(), ":v2:") {
		t.Fatalf("metadata key %q repeats the version outside its hash tag", topology.metadataKey())
	}
}

func TestSequenceQueueMessageStatusKeyUsesOrderKeyPartition(t *testing.T) {
	topology := newSequenceQueueTopology("prefix", "channel", "topic", 4)
	id := "message-01"
	orderKey := "order01"
	partition := topology.partition(orderKey)

	key := topology.messageStatusKey(partition, orderKey, id)
	if !strings.HasPrefix(key, "prefix:channel:topic:"+topology.partitionTag(partition)+":") {
		t.Fatalf("status key %q does not use order-key partition tag %q", key, topology.partitionTag(partition))
	}
	if !strings.HasSuffix(key, ":status:"+id) {
		t.Fatalf("status key %q does not include message id", key)
	}
	if firstRedisHashTag(key) != strings.Trim(topology.partitionTag(partition), "{}") {
		t.Fatalf("status key %q selected wrong Redis hash tag", key)
	}
}

func TestSequenceQueueTopologyKeysShareOnlyTheirPartitionSlot(t *testing.T) {
	topology := newSequenceQueueTopology("prefix", "channel", "topic", 4)
	partition0 := []string{
		topology.schedulerKey(0),
		topology.partitionStateKey(0),
		topology.isolationKey(0),
		topology.orderListKey(0, "order-a"),
		topology.orderStateKey(0, "order-b"),
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
