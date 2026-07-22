package bredis

import (
	"context"

	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/rs/xid"
)

type normalQueueRuntime struct {
	runtime *partitionRuntime[streamQueueDispatch]
}

func newNormalQueueRuntime(store *normalQueueStore, base *queueBase) *normalQueueRuntime {
	adapter := &streamQueueAdapter{
		name: "normal queue", consumerPrefix: "normal", instanceID: xid.New().String(),
		client: store.client, base: base, partitions: store.topology.partitions,
		ensureMetadata: store.ensureMetadata, bootstrapGroups: store.bootstrapGroups, bootstrapPartition: store.bootstrapGroup,
		streamKey: store.topology.streamKey, deadLetterLockKey: store.topology.deadLetterLockKey,
	}
	return &normalQueueRuntime{runtime: newPartitionRuntime[streamQueueDispatch](adapter)}
}

func (r *normalQueueRuntime) run(ctx context.Context, channel, topic string, handler public.CallbackWithRetry) {
	r.runtime.run(ctx, channel, topic, handler)
}
