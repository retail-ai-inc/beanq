package bredis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/spf13/cast"
)

type Schedule struct {
	base       queueBase
	maxLen     int64
	partitions int64
}

func NewSchedule(client redis.UniversalClient, prefix string, consumerCount int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config) *Schedule {
	return newScheduleWithPartitions(client, prefix, partitionQueueDefaultMaxLen, consumerCount, consumerPoolSize, deadLetterIdle, config)
}

func newScheduleWithPartitions(client redis.UniversalClient, prefix string, maxLen, partitions int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config) *Schedule {
	partitions = normalizePartitionCount(partitions)
	return &Schedule{
		maxLen: maxLen, partitions: partitions,
		base: newQueueBase(queueBaseOptions{client: client, prefix: prefix,
			deadLetterIdle: deadLetterIdle, consumerPoolSize: consumerPoolSize, captureConfig: config}),
	}
}

func (s *Schedule) store(channel, topic string) *delayQueueStore {
	return newDelayQueueStore(s.base.client, newDelayQueueTopology(s.base.prefix, channel, topic, s.partitions), s.maxLen)
}

func (s *Schedule) Publish(ctx context.Context, data map[string]any) error {
	store := s.store(cast.ToString(data["channel"]), cast.ToString(data["topic"]))
	if err := store.ensureMetadata(ctx); err != nil {
		return err
	}
	return store.enqueue(ctx, data)
}

func (s *Schedule) Consume(ctx context.Context, channel, topic string, handler public.CallbackWithRetry) {
	newDelayQueueRuntime(s.store(channel, topic), &s.base).run(ctx, channel, topic, handler)
}
