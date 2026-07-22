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

func TestResolveNormalQueuePartitions(t *testing.T) {
	if got := ResolveNormalQueuePartitions(12, 100); got != 12 {
		t.Fatalf("configured partitions = %d, want 12", got)
	}
	if got := ResolveNormalQueuePartitions(0, 7); got != 7 {
		t.Fatalf("fallback partitions = %d, want 7", got)
	}
}

func TestResolveQueuePartitions(t *testing.T) {
	if got := ResolveQueuePartitions(9, 7); got != 9 {
		t.Fatalf("configured partitions = %d, want 9", got)
	}
	if got := ResolveQueuePartitions(0, 7); got != 7 {
		t.Fatalf("fallback partitions = %d, want 7", got)
	}
}
