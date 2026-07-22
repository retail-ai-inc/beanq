package bredis

import (
	"strings"
	"testing"
)

func TestNormalQueueTopologyUsesStableMessageIDPartition(t *testing.T) {
	topology := newNormalQueueTopology("prefix", "channel", "topic", 13)
	for _, messageID := range []string{"message-a", "message-b", "message-c"} {
		partition := topology.partition(messageID)
		if partition < 0 || partition >= topology.partitions {
			t.Fatalf("partition(%q) = %d, out of range", messageID, partition)
		}
		if again := topology.partition(messageID); again != partition {
			t.Fatalf("partition(%q) changed from %d to %d", messageID, partition, again)
		}
	}
}

func TestNormalQueueTopologyProtectsClusterHashTag(t *testing.T) {
	topology := newNormalQueueTopology("prefix", "channel", "topic", 3)
	first := topology.streamKey(0)
	second := topology.streamKey(1)
	if !strings.HasPrefix(first, "prefix:channel:topic:"+topology.partitionTag(0)+":") {
		t.Fatalf("partition key does not start with configured route: %q", first)
	}
	if firstRedisHashTag(first) == firstRedisHashTag(second) {
		t.Fatalf("different partitions share hash tag %q", firstRedisHashTag(first))
	}
}

func TestNormalQueueCanonicalConfigIncludesPerPartitionCapacity(t *testing.T) {
	store := newNormalQueueStore(nil, newNormalQueueTopology("p", "c", "t", 7), 123)
	if got, want := store.canonicalConfig(), "schema=2;partitions=7;capacity=123"; got != want {
		t.Fatalf("canonicalConfig() = %q, want %q", got, want)
	}
}

func TestValidateNormalQueueMetadataRejectsMismatch(t *testing.T) {
	if err := validateNormalQueueMetadata("schema=2;partitions=2;capacity=10", "schema=2;partitions=3;capacity=10"); err == nil {
		t.Fatal("expected metadata mismatch")
	}
}
