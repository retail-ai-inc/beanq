package bredis

import (
	"testing"
	"time"

	"github.com/retail-ai-inc/beanq/v4/internal/btype"
)

func TestRdbBrokerSequenceQueueUsesFixedConfiguredPartitions(t *testing.T) {
	broker := NewBrokerWithSequenceQueuePartitions(nil, "prefix", 100, 7, 13, 2, time.Minute)
	if _, ok := broker.routes[btype.SEQUENCE_QUEUE]; !ok {
		t.Fatal("sequence queue route is not registered")
	}
	if consumer, err := broker.Consumer(btype.SEQUENCE_QUEUE, nil); err != nil || consumer == nil {
		t.Fatalf("sequence queue consumer = %#v, %v", consumer, err)
	}
}

func TestLegacyConstructorsKeepConsumerBasedSequenceQueueTopology(t *testing.T) {
	broker := NewBroker(nil, "prefix", 100, 7, 2, time.Minute)
	if _, err := broker.Consumer(btype.SEQUENCE_QUEUE, nil); err != nil {
		t.Fatalf("legacy broker sequence queue consumer: %v", err)
	}

	direct := NewSequenceQueue(nil, "prefix", 100, 9, 2, time.Minute, nil)
	if got := direct.partitions; got != 9 {
		t.Fatalf("direct constructor partition count = %d, want 9", got)
	}
}
