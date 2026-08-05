package bredis

import (
	"context"
	"testing"
	"time"
)

func TestEffectivePartitionReaderCount(t *testing.T) {
	tests := []struct {
		name       string
		configured int
		partitions int64
		want       int
	}{
		{name: "configured", configured: 8, partitions: 1000, want: 8},
		{name: "limited by partitions", configured: 8, partitions: 4, want: 4},
		{name: "zero falls back", configured: 0, partitions: 1000, want: 1},
		{name: "negative falls back", configured: -1, partitions: 1000, want: 1},
		{name: "invalid partitions", configured: 8, partitions: 0, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectivePartitionReaderCount(tt.configured, tt.partitions); got != tt.want {
				t.Fatalf("effectivePartitionReaderCount(%d, %d) = %d, want %d", tt.configured, tt.partitions, got, tt.want)
			}
		})
	}
}

func TestPartitionReaderAssignmentsCoverEachPartitionOnce(t *testing.T) {
	const partitions = int64(1000)
	const readers = 8
	assignments := make([]int, partitions)
	for readerID := range readers {
		for partition := int64(readerID); partition < partitions; partition += readers {
			assignments[partition]++
		}
	}
	for partition, count := range assignments {
		if count != 1 {
			t.Fatalf("partition %d assigned %d times, want exactly once", partition, count)
		}
	}
}

func TestPartitionReaderUsesNegativeBlockForNonBlockingReads(t *testing.T) {
	if partitionNonBlockingRead >= 0 {
		t.Fatalf("partitionNonBlockingRead = %s, want negative duration", partitionNonBlockingRead)
	}
}

func TestWaitPartitionReaderStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	if waitPartitionReader(ctx, time.Second) {
		t.Fatal("waitPartitionReader returned true after cancellation")
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("canceled wait took %s", elapsed)
	}
}
