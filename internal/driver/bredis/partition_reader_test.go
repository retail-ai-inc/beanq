package bredis

import (
	"context"
	"testing"
	"time"
)

func TestPartitionReaderUsesFixedConcurrency(t *testing.T) {
	if partitionReaderCount != 1 {
		t.Fatalf("partitionReaderCount = %d, want 1", partitionReaderCount)
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
