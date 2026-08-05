package bredis

import (
	"testing"

	"github.com/retail-ai-inc/beanq/v4/internal/btype"
)

func TestProcessLogStatusKeyUsesMessageIDQueueTopology(t *testing.T) {
	const (
		prefix  = "prefix"
		channel = "channel"
		topic   = "topic"
		id      = "message-01"
	)
	log := NewProcessLogWithPartitions(nil, prefix, 7, 11)
	data := map[string]any{"channel": channel, "topic": topic, "id": id}

	normalTopology := newNormalQueueTopology(prefix, channel, topic, 7)
	partition := normalTopology.partition(id)
	if got, want := log.statusKey(data, btype.NORMAL), normalTopology.messageStatusKey(partition, id); got != want {
		t.Fatalf("normal status key = %q, want %q", got, want)
	}

	delayTopology := newDelayQueueTopology(prefix, channel, topic, 7)
	if got, want := log.statusKey(data, btype.DELAY), delayTopology.messageStatusKey(partition, id); got != want {
		t.Fatalf("delay status key = %q, want %q", got, want)
	}
}

func TestProcessLogStatusKeyRequiresMessageID(t *testing.T) {
	log := NewProcessLogWithPartitions(nil, "prefix", 7, 11)
	if got := log.statusKey(map[string]any{"channel": "channel", "topic": "topic"}, btype.NORMAL); got != "" {
		t.Fatalf("status key without id = %q, want empty", got)
	}
}
