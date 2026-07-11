package bredis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	bjson "github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/rs/xid"
)

const sequenceQueueReadBlock = 500 * time.Millisecond

type sequenceQueueExecutor func(context.Context, string, string, string, public.CallbackWithRetry) (map[string]any, error)

type sequenceQueueRuntime struct {
	store      *sequenceQueueStore
	logger     processLogger
	execute    sequenceQueueExecutor
	workers    int
	partitions int64
	lease      time.Duration
	instanceID string
}

func newSequenceQueueRuntime(store *sequenceQueueStore, processLogger processLogger, workers int, execute sequenceQueueExecutor) *sequenceQueueRuntime {
	if workers <= 0 {
		workers = 1
	}
	return &sequenceQueueRuntime{
		store:      store,
		logger:     processLogger,
		execute:    execute,
		workers:    workers,
		partitions: store.topology.partitions,
		lease:      store.lease,
		instanceID: xid.New().String(),
	}
}

func (r *sequenceQueueRuntime) run(ctx context.Context, channel, topic string, handler public.CallbackWithRetry) {
	if err := r.store.ensureMetadata(ctx); err != nil {
		logger.New().Error(err)
		return
	}
	group := channel
	if err := r.store.bootstrapGroups(ctx, group); err != nil {
		logger.New().Error(err)
		return
	}

	var wait sync.WaitGroup
	wait.Add(r.workers)
	for workerID := 0; workerID < r.workers; workerID++ {
		go func(workerID int) {
			defer wait.Done()
			r.worker(ctx, channel, topic, group, workerID, handler)
		}(workerID)
	}
	wait.Wait()
}

func (r *sequenceQueueRuntime) worker(ctx context.Context, channel, topic, group string, workerID int, handler public.CallbackWithRetry) {
	consumer := fmt.Sprintf("sequence-queue-%s-%d", r.instanceID, workerID)
	owner := consumer
	offset := int64(workerID) % r.partitions
	claimEvery := r.heartbeatInterval()
	lastClaim := make([]time.Time, r.partitions)

	for ctx.Err() == nil {
		for scanned := int64(0); scanned < r.partitions && ctx.Err() == nil; scanned++ {
			partition := (offset + scanned) % r.partitions
			if lastClaim[partition].IsZero() || time.Since(lastClaim[partition]) >= claimEvery {
				lastClaim[partition] = time.Now()
				token, _, err := r.store.autoClaimOne(ctx, group, consumer, partition, r.lease)
				if err == nil {
					r.process(ctx, channel, topic, group, owner, consumer, *token, handler)
					continue
				}
				if !ignorableSequenceQueueReadError(ctx, err) {
					logger.New().Error(err)
				}
			}

			token, err := r.store.readOne(ctx, group, consumer, partition, sequenceQueueReadBlock)
			if err == nil {
				r.process(ctx, channel, topic, group, owner, consumer, *token, handler)
				continue
			}
			if !ignorableSequenceQueueReadError(ctx, err) {
				logger.New().Error(err)
			}
		}
		offset = (offset + 1) % r.partitions
	}
}

func (r *sequenceQueueRuntime) process(ctx context.Context, channel, topic, group, owner, consumer string, token sequenceQueueToken, handler public.CallbackWithRetry) {
	acquired, err := r.store.acquire(ctx, group, owner, consumer, token)
	if err != nil {
		logger.New().Error(err)
		return
	}
	if acquired.Code != sequenceQueueCodeAcquired {
		return
	}

	leaseCtx, cancelLease := context.WithCancel(ctx)
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		r.heartbeat(leaseCtx, cancelLease, group, owner, consumer, token)
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
		result = sequenceQueueFailedData(channel, topic, token.CustomerID, acquired.Head, executeErr)
	}
	if leaseCtx.Err() != nil {
		return
	}
	if err := r.logger.AddLog(leaseCtx, result); err != nil {
		logger.New().Error(err)
		return
	}

	finalized, err := r.store.finalize(leaseCtx, group, owner, consumer, token, acquired.Head)
	if err != nil {
		logger.New().Error(err)
		return
	}
	if finalized.Code != sequenceQueueCodeSuccessor && finalized.Code != sequenceQueueCodeEmpty {
		logger.New().Error(fmt.Errorf("sequence queue finalize rejected token %s: %s", token.SchedulerID, finalized.Code))
	}
}

func (r *sequenceQueueRuntime) heartbeat(ctx context.Context, cancelLease context.CancelFunc, group, owner, consumer string, token sequenceQueueToken) {
	ticker := time.NewTicker(r.heartbeatInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := r.store.heartbeat(ctx, group, owner, consumer, token)
			if err != nil {
				if ctx.Err() == nil {
					logger.New().Error(err)
					cancelLease()
				}
				return
			}
			if result.Code != sequenceQueueCodeHeartbeat {
				cancelLease()
				return
			}
		}
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
