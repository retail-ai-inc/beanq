package bredis

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	public "github.com/retail-ai-inc/beanq/v4/internal"
)

type testPartitionRuntimeAdapter struct {
	mu             sync.Mutex
	cursors        []string
	claimItems     []int
	readItems      []int
	processed      []int
	processStarted chan struct{}
	processRelease chan struct{}
	once           sync.Once
}

func (a *testPartitionRuntimeAdapter) Name() string                                  { return "test queue" }
func (a *testPartitionRuntimeAdapter) ConsumerPrefix() string                        { return "test" }
func (a *testPartitionRuntimeAdapter) InstanceID() string                            { return "instance" }
func (a *testPartitionRuntimeAdapter) Partitions() int64                             { return 1 }
func (a *testPartitionRuntimeAdapter) Workers() int                                  { return 1 }
func (a *testPartitionRuntimeAdapter) DispatchCapacity() int                         { return 1 }
func (a *testPartitionRuntimeAdapter) EnsureMetadata(context.Context) error          { return nil }
func (a *testPartitionRuntimeAdapter) BootstrapGroups(context.Context, string) error { return nil }
func (a *testPartitionRuntimeAdapter) BootstrapPartition(context.Context, string, int64) error {
	return nil
}
func (a *testPartitionRuntimeAdapter) StartBackground(context.Context, string, string) {}
func (a *testPartitionRuntimeAdapter) Claim(_ context.Context, _, _ string, _ int64, cursor string) ([]int, string, error) {
	a.mu.Lock()
	a.cursors = append(a.cursors, cursor)
	items := a.claimItems
	a.claimItems = nil
	a.mu.Unlock()
	return items, "7-0", nil
}
func (a *testPartitionRuntimeAdapter) Read(context.Context, string, string, int64) ([]int, error) {
	a.mu.Lock()
	items := a.readItems
	a.readItems = nil
	a.mu.Unlock()
	if len(items) == 0 {
		return nil, redis.Nil
	}
	return items, nil
}
func (a *testPartitionRuntimeAdapter) Process(ctx context.Context, _, _, _, _ string, item int, _ public.CallbackWithRetry) {
	a.once.Do(func() {
		if a.processStarted != nil {
			close(a.processStarted)
		}
	})
	a.mu.Lock()
	a.processed = append(a.processed, item)
	a.mu.Unlock()
	if a.processRelease != nil {
		<-a.processRelease
	}
}

func TestPartitionRuntimeCarriesClaimCursor(t *testing.T) {
	adapter := &testPartitionRuntimeAdapter{claimItems: []int{1}}
	runtime := newPartitionRuntime[int](adapter)
	dispatch := make(chan partitionDispatch[int], 2)
	state := &partitionReaderState{}

	if _, ok := runtime.pollPartition(context.Background(), "group", "consumer", 0, state, dispatch); !ok {
		t.Fatal("first poll stopped unexpectedly")
	}
	state.nextClaim = time.Time{}
	if _, ok := runtime.pollPartition(context.Background(), "group", "consumer", 0, state, dispatch); !ok {
		t.Fatal("second poll stopped unexpectedly")
	}

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if len(adapter.cursors) != 2 || adapter.cursors[0] != "" || adapter.cursors[1] != "7-0" {
		t.Fatalf("claim cursors = %#v, want [\"\" \"7-0\"]", adapter.cursors)
	}
}

func TestPartitionRuntimeCancellationDrainsWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	adapter := &testPartitionRuntimeAdapter{
		claimItems:     []int{1, 2, 3},
		processStarted: make(chan struct{}),
		processRelease: make(chan struct{}),
	}
	runtime := newPartitionRuntime[int](adapter)
	done := make(chan struct{})
	go func() {
		runtime.run(ctx, "channel", "topic", nil)
		close(done)
	}()

	select {
	case <-adapter.processStarted:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	cancel()
	select {
	case <-done:
		t.Fatal("runtime returned before in-flight worker completed")
	case <-time.After(20 * time.Millisecond):
	}
	close(adapter.processRelease)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runtime did not stop after cancellation")
	}
}

func TestStreamQueueBackgroundHookDoesNotBlockReaders(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	adapter := &streamQueueAdapter{
		base: &queueBase{},
		startExtra: func(ctx context.Context) {
			close(started)
			<-ctx.Done()
		},
	}
	returned := make(chan struct{})
	go func() {
		adapter.StartBackground(ctx, "channel", "topic")
		close(returned)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("background hook did not start")
	}
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("background hook blocked runtime startup")
	}
}
