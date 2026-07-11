package bredis

import (
	"testing"
	"time"

	"github.com/retail-ai-inc/beanq/v4/internal/btype"
)

func TestRdbBrokerSequenceQueueUsesFixedConfiguredPartitions(t *testing.T) {
	broker := NewBrokerWithSequenceQueuePartitions(nil, "prefix", 100, 7, 13, 2, time.Minute)
	queue, ok := broker.Mood(btype.SEQUENCE_QUEUE, nil).(*SequenceQueue)
	if !ok {
		t.Fatal("sequence queue mood did not return *SequenceQueue")
	}
	if got := queue.partitions; got != 13 {
		t.Fatalf("partition count = %d, want 13", got)
	}
}

func TestLegacyConstructorsKeepConsumerBasedSequenceQueueTopology(t *testing.T) {
	broker := NewBroker(nil, "prefix", 100, 7, 2, time.Minute)
	queue, ok := broker.Mood(btype.SEQUENCE_QUEUE, nil).(*SequenceQueue)
	if !ok {
		t.Fatal("sequence queue mood did not return *SequenceQueue")
	}
	if got := queue.partitions; got != 7 {
		t.Fatalf("partition count = %d, want legacy consumer count 7", got)
	}

	direct := NewSequenceQueue(nil, "prefix", 100, 9, 2, time.Minute, nil)
	if got := direct.partitions; got != 9 {
		t.Fatalf("direct constructor partition count = %d, want 9", got)
	}
}
