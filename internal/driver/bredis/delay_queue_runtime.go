package bredis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/rs/xid"
)

const (
	delaySchedulerConcurrency  = 8
	delaySchedulerInitialDelay = 100 * time.Millisecond
	delaySchedulerMaxDelay     = time.Second
)

type delayQueueRuntime struct {
	store   *delayQueueStore
	runtime *partitionRuntime[streamQueueDispatch]
}

type delayPartitionSchedule struct {
	next  time.Time
	delay time.Duration
}

type delayPromotionResult struct {
	partition int64
	promoted  bool
	err       error
}

func newDelayQueueRuntime(store *delayQueueStore, base *queueBase) *delayQueueRuntime {
	runtime := &delayQueueRuntime{store: store}
	adapter := &streamQueueAdapter{
		name: "delay queue", consumerPrefix: "delay", instanceID: xid.New().String(),
		client: store.client, base: base, partitions: store.topology.partitions,
		ensureMetadata: store.ensureMetadata, bootstrapGroups: store.bootstrapGroups, bootstrapPartition: store.bootstrapGroup,
		streamKey: store.topology.streamKey, deadLetterLockKey: store.topology.deadLetterLockKey,
		startExtra: runtime.scheduler,
	}
	runtime.runtime = newPartitionRuntime[streamQueueDispatch](adapter)
	return runtime
}

func (r *delayQueueRuntime) run(ctx context.Context, channel, topic string, handler public.CallbackWithRetry) {
	r.runtime.run(ctx, channel, topic, handler)
}

func (r *delayQueueRuntime) scheduler(ctx context.Context) {
	states := make([]delayPartitionSchedule, r.store.topology.partitions)
	for ctx.Err() == nil {
		now := time.Now()
		due := make([]int64, 0, len(states))
		for partition := range states {
			if states[partition].next.IsZero() || !now.Before(states[partition].next) {
				due = append(due, int64(partition))
			}
		}
		if len(due) == 0 {
			if !waitDelayScheduler(ctx, nextDelaySchedulerWake(states, now)) {
				return
			}
			continue
		}

		results := make(chan delayPromotionResult, len(due))
		for start := 0; start < len(due); start += delaySchedulerConcurrency {
			end := min(start+delaySchedulerConcurrency, len(due))
			var workers sync.WaitGroup
			for _, partition := range due[start:end] {
				partition := partition
				workers.Go(func() {
					promoted, err := r.store.promote(ctx, partition, now)
					results <- delayPromotionResult{partition: partition, promoted: promoted, err: err}
				})
			}
			workers.Wait()
		}
		close(results)

		active := false
		for result := range results {
			state := &states[result.partition]
			if result.err != nil && !errors.Is(result.err, context.Canceled) {
				logger.LogRuntimeError(ctx, fmt.Errorf("delay queue promote partition %d: %w", result.partition, result.err))
			}
			if result.promoted {
				state.reset()
				active = true
			} else {
				state.advance()
			}
		}
		if active {
			continue
		}
		if !waitDelayScheduler(ctx, nextDelaySchedulerWake(states, time.Now())) {
			return
		}
	}
}

func (s *delayPartitionSchedule) advance() {
	if s.delay <= 0 {
		s.delay = delaySchedulerInitialDelay
	} else {
		s.delay = min(s.delay*2, delaySchedulerMaxDelay)
	}
	jitterRange := max(s.delay/5, time.Nanosecond)
	jitter := time.Duration(time.Now().UnixNano() % int64(jitterRange))
	s.next = time.Now().Add(s.delay + jitter)
}

func (s *delayPartitionSchedule) reset() {
	s.delay = 0
	s.next = time.Time{}
}

func nextDelaySchedulerWake(states []delayPartitionSchedule, now time.Time) time.Duration {
	delay := delaySchedulerMaxDelay
	for _, state := range states {
		if state.next.IsZero() || !now.Before(state.next) {
			return 0
		}
		delay = min(delay, state.next.Sub(now))
	}
	return max(delay, time.Nanosecond)
}

func waitDelayScheduler(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
