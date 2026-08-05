package bredis

import (
	"strings"
	"testing"

	"github.com/retail-ai-inc/beanq/v4/internal/btype"
)

func TestDelayQueueTopologyUsesStableMessageIDPartition(t *testing.T) {
	topology := newDelayQueueTopology("prefix", "channel", "topic", 13)
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

func TestDelayQueuePartitionKeysShareOnlyTheirOwnSlot(t *testing.T) {
	topology := newDelayQueueTopology("prefix", "channel", "topic", 3)
	for partition := int64(0); partition < topology.partitions; partition++ {
		wantTag := firstRedisHashTag(topology.scheduledKey(partition))
		for _, key := range []string{
			topology.streamKey(partition),
			topology.deadLetterLockKey(partition),
		} {
			if got := firstRedisHashTag(key); got != wantTag {
				t.Fatalf("partition %d key %q uses hash tag %q, want %q", partition, key, got, wantTag)
			}
		}
	}
	if firstRedisHashTag(topology.scheduledKey(0)) == firstRedisHashTag(topology.scheduledKey(1)) {
		t.Fatal("different delay partitions share a Redis hash slot")
	}
	if !strings.HasPrefix(topology.scheduledKey(0), "prefix:channel:topic:"+topology.partitionTag(0)+":") {
		t.Fatal("delay key does not start with configured route")
	}
}

func TestDelayQueueMessageStatusKeyUsesMessageIDPartition(t *testing.T) {
	topology := newDelayQueueTopology("prefix", "channel", "topic", 7)
	id := "message-01"
	partition := topology.partition(id)
	key := topology.messageStatusKey(partition, id)
	if firstRedisHashTag(key) != firstRedisHashTag(topology.streamKey(partition)) {
		t.Fatalf("status key %q does not share the stream hash tag", key)
	}
	if !strings.HasSuffix(key, ":status:"+id) {
		t.Fatalf("status key = %q, want message status suffix", key)
	}
}

func TestDelayQueueCanonicalConfigUsesPerPartitionCapacity(t *testing.T) {
	store := newDelayQueueStore(nil, newDelayQueueTopology("p", "c", "t", 7), 123)
	if got, want := store.metadata().canonicalConfig(), "schema=2;partitions=7;capacity=123"; got != want {
		t.Fatalf("canonicalConfig() = %q, want %q", got, want)
	}
}

func TestBrokerDelayQueueReusesNormalPartitions(t *testing.T) {
	broker := NewBrokerWithPartitions(nil, "prefix", 100, 5, 11, 13, 2, 0)
	if _, ok := broker.routes[btype.DELAY]; !ok {
		t.Fatal("delay route is not registered")
	}
	delay := newScheduleWithPartitions(nil, "prefix", 100, 11, 2, 0, nil)
	if delay.partitions != 11 {
		t.Fatalf("delay partitions = %d, want normal queue partitions 11", delay.partitions)
	}
}
