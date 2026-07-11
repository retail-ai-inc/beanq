package boptions

import "testing"

func TestResolveSequenceQueuePartitions(t *testing.T) {
	tests := []struct {
		name         string
		partitions   int64
		minConsumers int64
		want         int64
	}{
		{name: "configured", partitions: 12, minConsumers: 100, want: 12},
		{name: "legacy fallback", partitions: 0, minConsumers: 7, want: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveSequenceQueuePartitions(tt.partitions, tt.minConsumers); got != tt.want {
				t.Fatalf("ResolveSequenceQueuePartitions(%d, %d) = %d, want %d", tt.partitions, tt.minConsumers, got, tt.want)
			}
		})
	}
}
