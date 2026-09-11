package logger

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestShutdownErrorCollectorAggregatesNormalizedErrors(t *testing.T) {
	collector := NewShutdownErrorCollector()
	collector.Add(errors.New("normal queue read partition 1: connection refused"))
	collector.Add(errors.New("normal queue read partition 2: connection refused"))

	collector.mu.Lock()
	defer collector.mu.Unlock()
	if len(collector.items) != 1 {
		t.Fatalf("items = %d, want 1", len(collector.items))
	}
	for _, item := range collector.items {
		if item.count != 2 {
			t.Fatalf("count = %d, want 2", item.count)
		}
	}
}

func TestShutdownErrorCollectorConcurrentAdd(t *testing.T) {
	collector := NewShutdownErrorCollector()
	var wait sync.WaitGroup
	for range 100 {
		wait.Go(func() { collector.Add(errors.New("redis connection refused")) })
	}
	wait.Wait()

	collector.mu.Lock()
	defer collector.mu.Unlock()
	for _, item := range collector.items {
		if item.count != 100 {
			t.Fatalf("count = %d, want 100", item.count)
		}
	}
}

func TestShutdownErrorCollectorIgnoresCancellation(t *testing.T) {
	collector := NewShutdownErrorCollector()
	collector.Add(context.Canceled)
	collector.Add(context.DeadlineExceeded)
	if len(collector.items) != 0 {
		t.Fatalf("items = %d, want 0", len(collector.items))
	}
}

func TestShutdownErrorCollectorFlushIsIdempotent(t *testing.T) {
	collector := NewShutdownErrorCollector()
	collector.Flush()
	collector.Flush()
	collector.Add(errors.New("late error"))
	if !collector.closed || len(collector.items) != 0 {
		t.Fatalf("collector state = closed:%v items:%d", collector.closed, len(collector.items))
	}
}
