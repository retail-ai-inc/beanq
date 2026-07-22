package bredis

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	public "github.com/retail-ai-inc/beanq/v4/internal"
)

type fakeSequenceQueueRuntimeStore struct {
	mu sync.Mutex

	ensureErr      error
	bootstrapErr   error
	read           func(context.Context, string, string, int64, time.Duration) (*sequenceQueueToken, error)
	claim          func(context.Context, string, string, int64, time.Duration, string, int64) (sequenceQueueAutoClaimResult, error)
	acquire        func(context.Context, string, string, string, sequenceQueueToken) (sequenceQueueAcquireResult, error)
	renewLease     func(context.Context, string, string, string, sequenceQueueToken) (sequenceQueueHeartbeatResult, error)
	finalize       func(context.Context, string, string, string, sequenceQueueToken, string) (sequenceQueueFinalizeResult, error)
	acquisitionIDs []string
	finalizeIDs    []string
}

func (s *fakeSequenceQueueRuntimeStore) ensureMetadata(context.Context) error { return s.ensureErr }
func (s *fakeSequenceQueueRuntimeStore) bootstrapGroups(context.Context, string) error {
	return s.bootstrapErr
}
func (s *fakeSequenceQueueRuntimeStore) readOne(ctx context.Context, group, consumer string, partition int64, block time.Duration) (*sequenceQueueToken, error) {
	return s.read(ctx, group, consumer, partition, block)
}
func (s *fakeSequenceQueueRuntimeStore) autoClaim(ctx context.Context, group, consumer string, partition int64, idle time.Duration, cursor string, count int64) (sequenceQueueAutoClaimResult, error) {
	return s.claim(ctx, group, consumer, partition, idle, cursor, count)
}
func (s *fakeSequenceQueueRuntimeStore) acquireWithID(ctx context.Context, group, consumer, acquisitionID string, token sequenceQueueToken) (sequenceQueueAcquireResult, error) {
	s.mu.Lock()
	s.acquisitionIDs = append(s.acquisitionIDs, acquisitionID)
	s.mu.Unlock()
	return s.acquire(ctx, group, consumer, acquisitionID, token)
}
func (s *fakeSequenceQueueRuntimeStore) renew(ctx context.Context, group, consumer, acquisitionID string, token sequenceQueueToken) (sequenceQueueHeartbeatResult, error) {
	return s.renewLease(ctx, group, consumer, acquisitionID, token)
}
func (s *fakeSequenceQueueRuntimeStore) finalizeWithID(ctx context.Context, group, consumer, acquisitionID string, token sequenceQueueToken, head string) (sequenceQueueFinalizeResult, error) {
	s.mu.Lock()
	s.finalizeIDs = append(s.finalizeIDs, acquisitionID)
	s.mu.Unlock()
	return s.finalize(ctx, group, consumer, acquisitionID, token, head)
}

type fakeSequenceQueueProcessLogger struct {
	mu   sync.Mutex
	logs []map[string]any
}

func (l *fakeSequenceQueueProcessLogger) AddLog(_ context.Context, data map[string]any) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, data)
	return nil
}

func TestSequenceQueueProcessUsesUniqueAcquisitionIDThroughFinalize(t *testing.T) {
	store := &fakeSequenceQueueRuntimeStore{}
	store.acquire = func(_ context.Context, _, _, acquisitionID string, _ sequenceQueueToken) (sequenceQueueAcquireResult, error) {
		return sequenceQueueAcquireResult{Code: bstatus.SequenceQueueCodeAcquired, Head: "payload", AcquisitionID: acquisitionID}, nil
	}
	store.renewLease = func(context.Context, string, string, string, sequenceQueueToken) (sequenceQueueHeartbeatResult, error) {
		return sequenceQueueHeartbeatResult{Code: bstatus.SequenceQueueCodeRenewed}, nil
	}
	store.finalize = func(_ context.Context, _, _, acquisitionID string, _ sequenceQueueToken, head string) (sequenceQueueFinalizeResult, error) {
		if head != "payload" || acquisitionID == "" {
			t.Fatalf("finalize head/id = %q/%q", head, acquisitionID)
		}
		return sequenceQueueFinalizeResult{Code: bstatus.SequenceQueueCodeEmpty}, nil
	}
	processLog := &fakeSequenceQueueProcessLogger{}
	runtime := &sequenceQueueRuntime{
		store: store, logger: processLog, workers: 1, partitions: 1,
		lease: time.Hour, instanceID: "test",
		execute: func(context.Context, string, string, string, public.CallbackWithRetry) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		},
	}
	token := sequenceQueueToken{Partition: 0, SchedulerID: "1-0", OrderKey: "order"}

	runtime.process(context.Background(), "channel", "topic", "channel", "consumer", token, nil)
	runtime.process(context.Background(), "channel", "topic", "channel", "consumer", token, nil)

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.acquisitionIDs) != 2 || len(store.finalizeIDs) != 2 {
		t.Fatalf("acquire/finalize calls = %d/%d, want 2/2", len(store.acquisitionIDs), len(store.finalizeIDs))
	}
	if store.acquisitionIDs[0] == store.acquisitionIDs[1] {
		t.Fatalf("acquisition ID was reused: %q", store.acquisitionIDs[0])
	}
	for i := range store.acquisitionIDs {
		if store.finalizeIDs[i] != store.acquisitionIDs[i] {
			t.Fatalf("finalize ID %q does not match acquire ID %q", store.finalizeIDs[i], store.acquisitionIDs[i])
		}
	}
}

func TestSequenceQueuePollPartitionClaimsThenReads(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	var operations []string
	store := &fakeSequenceQueueRuntimeStore{}
	store.claim = func(_ context.Context, _, _ string, partition int64, _ time.Duration, cursor string, count int64) (sequenceQueueAutoClaimResult, error) {
		mu.Lock()
		operations = append(operations, "claim:"+cursor)
		mu.Unlock()
		if partition != 2 || count != sequenceQueueAutoClaimBatch {
			t.Errorf("claim partition/count = %d/%d", partition, count)
		}
		return sequenceQueueAutoClaimResult{Cursor: "7-0", Tokens: []sequenceQueueToken{{Partition: 2, SchedulerID: "1-0", OrderKey: "key"}}}, nil
	}
	store.read = func(ctx context.Context, _, _ string, partition int64, _ time.Duration) (*sequenceQueueToken, error) {
		mu.Lock()
		operations = append(operations, "read")
		mu.Unlock()
		cancel()
		return nil, ctx.Err()
	}
	runtime := &sequenceQueueRuntime{store: store, workers: 1, partitions: 3, lease: time.Minute, instanceID: "test"}
	adapter := &sequenceQueueRuntimeAdapter{runtime: runtime}
	partitionRuntime := newPartitionRuntime[sequenceQueueToken](adapter)
	dispatch := make(chan partitionDispatch[sequenceQueueToken], 1)
	done := make(chan struct{})
	state := &partitionReaderState{}
	go func() {
		partitionRuntime.pollPartition(ctx, "group", "sequence-queue-test-r0", 2, state, dispatch)
		close(done)
	}()

	select {
	case item := <-dispatch:
		if item.item.SchedulerID != "1-0" || item.consumer != "sequence-queue-test-r0" {
			t.Fatalf("unexpected dispatch: %#v", item)
		}
	case <-time.After(time.Second):
		t.Fatal("partition poll did not dispatch claimed token")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("partition poll did not stop after cancellation")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(operations) != 2 || operations[0] != "claim:0-0" || operations[1] != "read" {
		t.Fatalf("operations = %#v, want claim/read", operations)
	}
}

func TestSequenceQueueRuntimeStopsAllPumpsAndWorkers(t *testing.T) {
	store := &fakeSequenceQueueRuntimeStore{}
	store.claim = func(ctx context.Context, _ string, _ string, _ int64, _ time.Duration, _ string, _ int64) (sequenceQueueAutoClaimResult, error) {
		<-ctx.Done()
		return sequenceQueueAutoClaimResult{}, ctx.Err()
	}
	store.read = func(ctx context.Context, _ string, _ string, _ int64, _ time.Duration) (*sequenceQueueToken, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	store.acquire = func(context.Context, string, string, string, sequenceQueueToken) (sequenceQueueAcquireResult, error) {
		return sequenceQueueAcquireResult{}, nil
	}
	store.renewLease = func(context.Context, string, string, string, sequenceQueueToken) (sequenceQueueHeartbeatResult, error) {
		return sequenceQueueHeartbeatResult{}, nil
	}
	store.finalize = func(context.Context, string, string, string, sequenceQueueToken, string) (sequenceQueueFinalizeResult, error) {
		return sequenceQueueFinalizeResult{}, nil
	}
	runtime := &sequenceQueueRuntime{
		store: store, logger: &fakeSequenceQueueProcessLogger{}, workers: 2, partitions: 4,
		lease: time.Minute, instanceID: "test",
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runtime.run(ctx, "channel", "topic", nil)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runtime did not stop pumps and workers after cancellation")
	}
}

func TestClassifySequenceQueueStoreCode(t *testing.T) {
	cases := map[string]sequenceQueueStoreState{
		bstatus.SequenceQueueCodeAcquired:         sequenceQueueStoreNormal,
		bstatus.SequenceQueueCodeRenewed:          sequenceQueueStoreNormal,
		bstatus.SequenceQueueCodeSuccessor:        sequenceQueueStoreNormal,
		bstatus.SequenceQueueCodeEmpty:            sequenceQueueStoreNormal,
		bstatus.SequenceQueueCodeBusy:             sequenceQueueStoreNormal,
		bstatus.SequenceQueueCodeStaleAcquisition: sequenceQueueStoreStaleOrCleaned,
		bstatus.SequenceQueueCodeStaleCleaned:     sequenceQueueStoreStaleOrCleaned,
		bstatus.SequenceQueueCodeOrphanCleaned:    sequenceQueueStoreStaleOrCleaned,
		bstatus.SequenceQueueCodeEmptyCleaned:     sequenceQueueStoreStaleOrCleaned,
		bstatus.SequenceQueueCodeNotPending:       sequenceQueueStoreStaleOrCleaned,
		bstatus.SequenceQueueCodePELOwnerMismatch: sequenceQueueStoreStaleOrCleaned,
		bstatus.SequenceQueueCodeIsolated:         sequenceQueueStoreIsolated,
		bstatus.SequenceQueueCodeHeadMismatch:     sequenceQueueStoreIsolated,
		"BAD_IDENTITY":                            sequenceQueueStoreFatal,
	}
	for code, want := range cases {
		if got := classifySequenceQueueStoreCode(code); got != want {
			t.Errorf("classify(%q) = %s, want %s", code, got, want)
		}
	}
}
