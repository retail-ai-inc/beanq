package bredis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/spf13/cast"
)

type Normal struct {
	base       queueBase
	maxLen     int64
	partitions int64
	wait       replicationWait
}

func NewNormal(client redis.UniversalClient, prefix string, maxLen int64, consumerCount int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config) *Normal {
	return newNormalWithPartitions(client, prefix, maxLen, consumerCount, consumerPoolSize, deadLetterIdle, config)
}

func newNormalWithPartitions(client redis.UniversalClient, prefix string, maxLen, partitions int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config) *Normal {
	return newNormalWithPartitionsAndWait(client, prefix, maxLen, partitions, consumerPoolSize, deadLetterIdle, config, replicationWait{})
}

func newNormalWithPartitionsAndWait(client redis.UniversalClient, prefix string, maxLen, partitions int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config, wait replicationWait) *Normal {
	return &Normal{
		maxLen: maxLen, partitions: normalizeSequenceQueuePartitionCount(partitions), wait: wait,
		base: newQueueBase(queueBaseOptions{client: client, prefix: prefix,
			deadLetterIdle: deadLetterIdle, consumerPoolSize: consumerPoolSize, captureConfig: config, wait: wait}),
	}
}

func (t *Normal) store(channel, topic string) *normalQueueStore {
	return newNormalQueueStore(t.base.client, newNormalQueueTopology(t.base.prefix, channel, topic, t.partitions), t.maxLen, t.wait)
}

func (t *Normal) Publish(ctx context.Context, data map[string]any) error {
	channel, topic := cast.ToString(data["channel"]), cast.ToString(data["topic"])
	messageID := cast.ToString(data["id"])
	if messageID == "" {
		return errors.New("normal queue message id is required")
	}
	store := t.store(channel, topic)
	if err := store.ensureMetadata(ctx); err != nil {
		return err
	}
	partition := store.topology.partition(messageID)
	args := NewZAddArgs(store.topology.streamKey(partition), "", "*", store.maxLen, 0, data)
	if !t.wait.enabled() {
		if err := t.base.client.XAdd(ctx, args).Err(); err != nil {
			return fmt.Errorf("[RedisBroker.enqueue] normal partition %d xadd error: %w", partition, err)
		}
		return nil
	}
	stream := store.topology.streamKey(partition)
	_, err := t.wait.execute(ctx, t.base.client, stream, func(pipe redis.Pipeliner) redis.Cmder {
		return pipe.XAdd(ctx, args)
	})
	if err != nil {
		return fmt.Errorf("[RedisBroker.enqueue] normal partition %d xadd error: %w", partition, err)
	}
	return nil
}

func (t *Normal) Consume(ctx context.Context, channel, topic string, do public.CallbackWithRetry) {
	newNormalQueueRuntime(t.store(channel, topic), &t.base).run(ctx, channel, topic, do)
}
