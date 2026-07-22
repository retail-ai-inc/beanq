package bredis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/rs/xid"
)

type delayQueueRuntime struct {
	store   *delayQueueStore
	runtime *partitionRuntime[streamQueueDispatch]
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
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		active := false
		now := time.Now()
		for partition := int64(0); partition < r.store.topology.partitions; partition++ {
			promoted, err := r.store.promote(ctx, partition, now)
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.New().Error(fmt.Errorf("delay queue promote partition %d: %w", partition, err))
			}
			active = active || promoted
		}
		if active {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
