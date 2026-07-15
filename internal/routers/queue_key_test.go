package routers

import "testing"

func TestPartitionedQueueRoutePreservesChannelAndTopic(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		channel string
		topic   string
		mood    string
	}{
		{"normal", "beanq:orders-channel:created-topic:{beanq-normal-v2:queuehash:p000}:normal_queue:stream000", "orders-channel", "created-topic", "normal"},
		{"delay", "beanq:delay-channel:retry-topic:{beanq-delay-v2:queuehash:p001}:delay_queue:stream001", "delay-channel", "retry-topic", "delay"},
		{"sequence queue", "beanq:order-channel:order-topic:{beanq-sq-v3:queuehash:p002}:sequence_queue:stream002:scheduler", "order-channel", "order-topic", "sequence_queue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, ok := partitionedQueueRoute(tt.key, "beanq")
			if !ok {
				t.Fatal("expected partitioned queue key to be recognized")
			}
			if stream.Channel != tt.channel || stream.Topic != tt.topic {
				t.Fatalf("route = %q/%q, want %q/%q", stream.Channel, stream.Topic, tt.channel, tt.topic)
			}
			if stream.MoodType != tt.mood {
				t.Fatalf("moodType = %q, want %q", stream.MoodType, tt.mood)
			}
		})
	}
}

func TestPartitionedQueueRouteIgnoresAuxiliaryKeys(t *testing.T) {
	keys := []string{
		"beanq:channel:topic:{beanq-normal-v2:queuehash:p000}:normal_queue:stream000:dead_letter_lock",
		"beanq:channel:topic:{beanq-sq-v3:queuehash:p000}:sequence_queue:stream000:state",
		"beanq:channel:topic:{beanq-sq-v3:queuehash:p000}:sequence_queue:stream000:order:digest:list",
	}
	for _, key := range keys {
		if _, ok := partitionedQueueRoute(key, "beanq"); ok {
			t.Fatalf("auxiliary key must not be reported as a queue stream: %q", key)
		}
	}
}

func TestLegacyQueueMoodType(t *testing.T) {
	tests := map[string]string{
		"normal_stream": "normal",
		"delay_stream":  "delay",
	}
	for input, want := range tests {
		if got := legacyQueueMoodType(input); got != want {
			t.Fatalf("legacyQueueMoodType(%q) = %q, want %q", input, got, want)
		}
	}
}
