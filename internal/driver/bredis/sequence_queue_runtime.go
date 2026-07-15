package bredis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	bjson "github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/rs/xid"
)

const (
	sequenceQueueAutoClaimBatch = int64(16)
)

type sequenceQueueExecutor func(context.Context, string, string, string, public.CallbackWithRetry) (map[string]any, error)

type sequenceQueueRuntimeStore interface {
	ensureMetadata(context.Context) error
	bootstrapGroups(context.Context, string) error
	readOne(context.Context, string, string, int64, time.Duration) (*sequenceQueueToken, error)
	autoClaim(context.Context, string, string, int64, time.Duration, string, int64) (sequenceQueueAutoClaimResult, error)
	acquireWithID(context.Context, string, string, string, sequenceQueueToken) (sequenceQueueAcquireResult, error)
	renew(context.Context, string, string, string, sequenceQueueToken) (sequenceQueueHeartbeatResult, error)
	finalizeWithID(context.Context, string, string, string, sequenceQueueToken, string) (sequenceQueueFinalizeResult, error)
}

type sequenceQueueRuntime struct {
	store      sequenceQueueRuntimeStore
	logger     processLogger
	execute    sequenceQueueExecutor
	workers    int
	partitions int64
	lease      time.Duration
	instanceID string
	runtime    *partitionRuntime[sequenceQueueToken]
}

type sequenceQueueRuntimeAdapter struct {
	runtime *sequenceQueueRuntime
}

type sequenceQueueStoreState uint8

const (
	sequenceQueueStoreNormal sequenceQueueStoreState = iota
	sequenceQueueStoreStaleOrCleaned
	sequenceQueueStoreIsolated
	sequenceQueueStoreFatal
)

func newSequenceQueueRuntime(store *sequenceQueueStore, processLogger processLogger, workers int, execute sequenceQueueExecutor) *sequenceQueueRuntime {
	if workers <= 0 {
		workers = 1
	}
	runtime := &sequenceQueueRuntime{
		store:      store,
		logger:     processLogger,
		execute:    execute,
		workers:    workers,
		partitions: store.topology.partitions,
		lease:      store.lease,
		instanceID: xid.New().String(),
	}
	runtime.runtime = newPartitionRuntime[sequenceQueueToken](&sequenceQueueRuntimeAdapter{runtime: runtime})
	return runtime
}

func (r *sequenceQueueRuntime) run(ctx context.Context, channel, topic string, handler public.CallbackWithRetry) {
	if r.runtime == nil {
		r.runtime = newPartitionRuntime[sequenceQueueToken](&sequenceQueueRuntimeAdapter{runtime: r})
	}
	r.runtime.run(ctx, channel, topic, handler)
}

func (a *sequenceQueueRuntimeAdapter) Name() string           { return "sequence queue" }
func (a *sequenceQueueRuntimeAdapter) ConsumerPrefix() string { return "sequence-queue" }
func (a *sequenceQueueRuntimeAdapter) InstanceID() string     { return a.runtime.instanceID }
func (a *sequenceQueueRuntimeAdapter) Partitions() int64      { return a.runtime.partitions }
func (a *sequenceQueueRuntimeAdapter) Workers() int           { return a.runtime.workers }
func (a *sequenceQueueRuntimeAdapter) DispatchCapacity() int  { return a.runtime.workers }
func (a *sequenceQueueRuntimeAdapter) EnsureMetadata(ctx context.Context) error {
	return a.runtime.store.ensureMetadata(ctx)
}
func (a *sequenceQueueRuntimeAdapter) BootstrapGroups(ctx context.Context, group string) error {
	return a.runtime.store.bootstrapGroups(ctx, group)
}
func (a *sequenceQueueRuntimeAdapter) BootstrapPartition(ctx context.Context, group string, _ int64) error {
	return a.runtime.store.bootstrapGroups(ctx, group)
}
func (a *sequenceQueueRuntimeAdapter) StartBackground(context.Context, string, string) {}
func (a *sequenceQueueRuntimeAdapter) Claim(ctx context.Context, group, consumer string, partition int64, cursor string) ([]sequenceQueueToken, string, error) {
	if cursor == "" {
		cursor = "0-0"
	}
	claimed, err := a.runtime.store.autoClaim(ctx, group, consumer, partition, a.runtime.lease, cursor, sequenceQueueAutoClaimBatch)
	return claimed.Tokens, claimed.Cursor, err
}
func (a *sequenceQueueRuntimeAdapter) Read(ctx context.Context, group, consumer string, partition int64) ([]sequenceQueueToken, error) {
	token, err := a.runtime.store.readOne(ctx, group, consumer, partition, partitionNonBlockingRead)
	if err != nil {
		return nil, err
	}
	return []sequenceQueueToken{*token}, nil
}
func (a *sequenceQueueRuntimeAdapter) Process(ctx context.Context, channel, topic, group, consumer string, token sequenceQueueToken, handler public.CallbackWithRetry) {
	a.runtime.process(ctx, channel, topic, group, consumer, token, handler)
}

func (r *sequenceQueueRuntime) process(ctx context.Context, channel, topic, group, consumer string, token sequenceQueueToken, handler public.CallbackWithRetry) {
	acquisitionID := xid.New().String()
	acquired, err := r.store.acquireWithID(ctx, group, consumer, acquisitionID, token)
	if err != nil {
		logger.New().Error(err)
		return
	}
	if state := classifySequenceQueueStoreCode(acquired.Code); state != sequenceQueueStoreNormal {
		r.recordStoreState("acquire", token, acquired.Code, state)
		return
	}
	if acquired.Code != bstatus.SequenceQueueCodeAcquired {
		r.recordStoreState("acquire", token, acquired.Code, sequenceQueueStoreFatal)
		return
	}

	leaseCtx, cancelLease := context.WithCancel(ctx)
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		r.heartbeat(leaseCtx, cancelLease, group, consumer, acquisitionID, token)
	}()
	defer func() {
		cancelLease()
		<-heartbeatDone
	}()

	result, executeErr := r.execute(leaseCtx, channel, topic, acquired.Head, handler)
	if executeErr != nil && result == nil {
		if errors.Is(executeErr, context.Canceled) {
			return
		}
		result = sequenceQueueFailedData(channel, topic, token.OrderKey, acquired.Head, executeErr)
	}
	if leaseCtx.Err() != nil {
		return
	}
	if err := r.logger.AddLog(leaseCtx, result); err != nil {
		logger.New().Error(err)
		return
	}

	finalized, err := r.store.finalizeWithID(leaseCtx, group, consumer, acquisitionID, token, acquired.Head)
	if err != nil {
		logger.New().Error(err)
		return
	}
	state := classifySequenceQueueStoreCode(finalized.Code)
	if state != sequenceQueueStoreNormal || (finalized.Code != bstatus.SequenceQueueCodeSuccessor && finalized.Code != bstatus.SequenceQueueCodeEmpty) {
		if state == sequenceQueueStoreNormal {
			state = sequenceQueueStoreFatal
		}
		r.recordStoreState("finalize", token, finalized.Code, state)
	}
}

func (r *sequenceQueueRuntime) heartbeat(ctx context.Context, cancelLease context.CancelFunc, group, consumer, acquisitionID string, token sequenceQueueToken) {
	ticker := time.NewTicker(r.heartbeatInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := r.store.renew(ctx, group, consumer, acquisitionID, token)
			if err != nil {
				if ctx.Err() == nil {
					logger.New().Error(err)
					cancelLease()
				}
				return
			}
			state := classifySequenceQueueStoreCode(result.Code)
			if state != sequenceQueueStoreNormal || result.Code != bstatus.SequenceQueueCodeRenewed {
				if state == sequenceQueueStoreNormal {
					state = sequenceQueueStoreFatal
				}
				r.recordStoreState("renew", token, result.Code, state)
				cancelLease()
				return
			}
		}
	}
}

func classifySequenceQueueStoreCode(code string) sequenceQueueStoreState {
	switch code {
	case bstatus.SequenceQueueCodeAcquired, bstatus.SequenceQueueCodeRenewed, bstatus.SequenceQueueCodeSuccessor,
		bstatus.SequenceQueueCodeEmpty, bstatus.SequenceQueueCodeBusy:
		return sequenceQueueStoreNormal
	case bstatus.SequenceQueueCodeStaleAcquisition, bstatus.SequenceQueueCodeStaleCleaned,
		bstatus.SequenceQueueCodeOrphanCleaned, bstatus.SequenceQueueCodeEmptyCleaned,
		bstatus.SequenceQueueCodeNotPending, bstatus.SequenceQueueCodePELOwnerMismatch:
		return sequenceQueueStoreStaleOrCleaned
	case bstatus.SequenceQueueCodeIsolated, bstatus.SequenceQueueCodeHeadMismatch:
		return sequenceQueueStoreIsolated
	default:
		return sequenceQueueStoreFatal
	}
}

func (r *sequenceQueueRuntime) recordStoreState(operation string, token sequenceQueueToken, code string, state sequenceQueueStoreState) {
	err := fmt.Errorf("sequence queue %s token %s order key %s returned %s (state=%s)", operation, token.SchedulerID, token.OrderKey, code, state)
	if state == sequenceQueueStoreStaleOrCleaned {
		logger.New().Info(err)
		return
	}
	logger.New().Error(err)
}

func (s sequenceQueueStoreState) String() string {
	switch s {
	case sequenceQueueStoreNormal:
		return "normal"
	case sequenceQueueStoreStaleOrCleaned:
		return "stale/cleaned"
	case sequenceQueueStoreIsolated:
		return "isolated"
	default:
		return "fatal"
	}
}

func (r *sequenceQueueRuntime) heartbeatInterval() time.Duration {
	interval := r.lease / 3
	if interval <= 0 {
		return time.Second
	}
	return interval
}

func ignorableSequenceQueueReadError(ctx context.Context, err error) bool {
	return errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil
}

func executeSequenceQueueMessage(ctx context.Context, channel, topic, raw string, handler public.CallbackWithRetry, config *capture.Config) (map[string]any, error) {
	var data map[string]any
	if err := bjson.Unmarshal([]byte(raw), &data); err != nil {
		return nil, err
	}
	result, ok := executeMessage(ctx, public.Stream{Data: data, Channel: channel, Stream: topic}, handler, config)
	if !ok {
		return nil, context.Canceled
	}
	return result.Data, nil
}
