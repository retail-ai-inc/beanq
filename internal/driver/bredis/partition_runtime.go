package bredis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/boptions"
)

type partitionRuntimeAdapter[T any] interface {
	Name() string
	ConsumerPrefix() string
	Partitions() int64
	Workers() int
	Readers() int
	DispatchCapacity() int
	EnsureMetadata(context.Context) error
	BootstrapGroups(context.Context, string) error
	BootstrapPartition(context.Context, string, int64) error
	StartBackground(context.Context, string, string)
	Claim(context.Context, string, string, int64, string) ([]T, string, error)
	Read(context.Context, string, string, int64) ([]T, error)
	Process(context.Context, string, string, string, string, T, public.CallbackWithRetry)
}

type partitionRuntime[T any] struct {
	adapter partitionRuntimeAdapter[T]
}

type partitionReaderState struct {
	cursor    string
	nextClaim time.Time
	nextRead  time.Time
	idleDelay time.Duration
}

func newPartitionRuntime[T any](adapter partitionRuntimeAdapter[T]) *partitionRuntime[T] {
	return &partitionRuntime[T]{adapter: adapter}
}

func (r *partitionRuntime[T]) run(ctx context.Context, channel, topic string, handler public.CallbackWithRetry) {
	if err := r.adapter.EnsureMetadata(ctx); err != nil {
		r.logError(ctx, "initialize metadata", -1, err)
		return
	}
	if err := r.adapter.BootstrapGroups(ctx, channel); err != nil {
		r.logError(ctx, "bootstrap groups", -1, err)
		return
	}

	workers := r.adapter.Workers()
	if workers <= 0 {
		workers = 1
	}
	capacity := r.adapter.DispatchCapacity()
	if capacity <= 0 {
		capacity = workers
	}
	dispatch := make(chan partitionDispatch[T], capacity)
	processingCtx, cancelProcessing := gracefulProcessingContext(ctx, r.gracefulShutdownTimeout())
	defer cancelProcessing()

	var workerWait sync.WaitGroup
	for range workers {
		workerWait.Go(func() { r.worker(processingCtx, channel, topic, dispatch, handler) })
	}
	r.adapter.StartBackground(ctx, channel, topic)

	readerCount := effectivePartitionReaderCount(r.adapter.Readers(), r.adapter.Partitions())
	var readerWait sync.WaitGroup
	for readerID := range readerCount {
		readerID := readerID
		readerWait.Go(func() { r.reader(ctx, channel, readerID, readerCount, dispatch) })
	}

	<-ctx.Done()
	readerWait.Wait()
	close(dispatch)
	workerWait.Wait()
}

func (r *partitionRuntime[T]) gracefulShutdownTimeout() time.Duration {
	if provider, ok := r.adapter.(interface{ GracefulShutdownTimeout() time.Duration }); ok {
		return provider.GracefulShutdownTimeout()
	}
	return boptions.DefaultGracefulShutdownTimeout
}

type partitionDispatch[T any] struct {
	consumer string
	item     T
}

func effectivePartitionReaderCount(configured int, partitions int64) int {
	if configured <= 0 {
		configured = 1
	}
	if partitions <= 0 {
		return 1
	}
	if int64(configured) > partitions {
		return int(partitions)
	}
	return configured
}

func (r *partitionRuntime[T]) reader(ctx context.Context, group string, readerID, readerCount int, dispatch chan<- partitionDispatch[T]) {
	consumer := fmt.Sprintf("%s-%s-r%d", r.adapter.ConsumerPrefix(), xidForRuntime(r.adapter), readerID)
	states := make([]partitionReaderState, r.adapter.Partitions())
	for ctx.Err() == nil {
		active := false
		for partition := int64(readerID); partition < r.adapter.Partitions(); partition += int64(readerCount) {
			partitionActive, ok := r.pollPartition(ctx, group, consumer, partition, &states[partition], dispatch)
			if !ok {
				return
			}
			active = active || partitionActive
		}
		if !active && !waitPartitionReader(ctx, partitionReaderIdleDelay) {
			return
		}
	}
}

func (r *partitionRuntime[T]) pollPartition(ctx context.Context, group, consumer string, partition int64, state *partitionReaderState, dispatch chan<- partitionDispatch[T]) (bool, bool) {
	active := false
	if !time.Now().Before(state.nextClaim) {
		state.nextClaim = time.Now().Add(partitionClaimInterval)
		items, cursor, err := r.adapter.Claim(ctx, group, consumer, partition, state.cursor)
		state.cursor = cursor
		if err != nil {
			if !ignorablePartitionReadError(ctx, err) {
				r.logError(ctx, "autoclaim", partition, err)
			}
		} else {
			active = len(items) > 0
			for _, item := range items {
				if !sendPartitionDispatch(ctx, dispatch, partitionDispatch[T]{consumer: consumer, item: item}) {
					return active, false
				}
			}
		}
	}
	if active {
		state.resetReadBackoff()
	}
	if time.Now().Before(state.nextRead) {
		return active, ctx.Err() == nil
	}

	items, err := r.adapter.Read(ctx, group, consumer, partition)
	if err != nil {
		if stringsContainsNoGroup(err) {
			if groupErr := r.adapter.BootstrapPartition(ctx, group, partition); groupErr != nil {
				r.logError(ctx, "bootstrap partition", partition, groupErr)
			}
			return active, true
		}
		if !ignorablePartitionReadError(ctx, err) {
			r.logError(ctx, "read", partition, err)
		}
		if errors.Is(err, redis.Nil) {
			state.advanceReadBackoff()
		}
		return active, ctx.Err() == nil
	}
	active = active || len(items) > 0
	if len(items) == 0 {
		state.advanceReadBackoff()
	} else {
		state.resetReadBackoff()
	}
	for _, item := range items {
		if !sendPartitionDispatch(ctx, dispatch, partitionDispatch[T]{consumer: consumer, item: item}) {
			return active, false
		}
	}
	return active, ctx.Err() == nil
}

func (s *partitionReaderState) advanceReadBackoff() {
	if s.idleDelay <= 0 {
		s.idleDelay = partitionReaderIdleDelay
	} else {
		s.idleDelay = min(s.idleDelay*2, partitionReaderMaxDelay)
	}
	jitterRange := max(s.idleDelay/5, time.Nanosecond)
	jitter := time.Duration(time.Now().UnixNano() % int64(jitterRange))
	s.nextRead = time.Now().Add(s.idleDelay + jitter)
}

func (s *partitionReaderState) resetReadBackoff() {
	s.idleDelay = 0
	s.nextRead = time.Time{}
}

func (r *partitionRuntime[T]) worker(ctx context.Context, channel, topic string, dispatch <-chan partitionDispatch[T], handler public.CallbackWithRetry) {
	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-dispatch:
			if !ok {
				return
			}
			r.adapter.Process(ctx, channel, topic, channel, item.consumer, item.item, handler)
		}
	}
}

func (r *partitionRuntime[T]) logError(ctx context.Context, operation string, partition int64, err error) {
	if partition >= 0 {
		logger.LogRuntimeError(ctx, fmt.Errorf("%s %s partition %d: %w", r.adapter.Name(), operation, partition, err))
		return
	}
	logger.LogRuntimeError(ctx, fmt.Errorf("%s %s: %w", r.adapter.Name(), operation, err))
}

func sendPartitionDispatch[T any](ctx context.Context, dispatch chan<- partitionDispatch[T], item partitionDispatch[T]) bool {
	select {
	case <-ctx.Done():
		return false
	case dispatch <- item:
		return true
	}
}

func ignorablePartitionReadError(ctx context.Context, err error) bool {
	return errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil
}

func stringsContainsNoGroup(err error) bool {
	return err != nil && strings.Contains(err.Error(), "NOGROUP")
}

type runtimeIdentity interface {
	InstanceID() string
}

func xidForRuntime(adapter any) string {
	if identity, ok := adapter.(runtimeIdentity); ok {
		return identity.InstanceID()
	}
	return "runtime"
}
