package bredis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	bjson "github.com/retail-ai-inc/beanq/v4/helper/json"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/spf13/cast"
)

type SequenceQueue struct {
	base       Base
	maxLen     int64
	partitions int64
}

func NewSequenceQueue(client redis.UniversalClient, prefix string, maxLen int64, consumerCount int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config) *SequenceQueue {
	return newSequenceQueueWithPartitions(client, prefix, maxLen, consumerCount, consumerPoolSize, deadLetterIdle, config)
}

func newSequenceQueueWithPartitions(client redis.UniversalClient, prefix string, maxLen, partitions int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config) *SequenceQueue {
	return &SequenceQueue{
		maxLen:     maxLen,
		partitions: normalizeSequenceQueuePartitionCount(partitions),
		base: Base{
			client:           client,
			processLogger:    NewProcessLog(client, prefix),
			prefix:           prefix,
			deadLetterIdle:   deadLetterIdle,
			consumerPoolSize: consumerPoolSize,
			captureConfig:    config,
		},
	}
}

func (q *SequenceQueue) Enqueue(ctx context.Context, data map[string]any) error {
	return q.PublishNewSequence(ctx, data)
}

func (q *SequenceQueue) Dequeue(ctx context.Context, channel, topic string, do public.CallbackWithRetry) {
	q.ConsumerSequence(ctx, channel, topic, do)
}

func (q *SequenceQueue) PublishNewSequence(ctx context.Context, data map[string]any) error {
	channel, topic, customerID := sequenceQueueMessageRoute(data)
	if customerID == "" {
		return errors.New("missing customerId")
	}
	store := q.sequenceQueueStore(channel, topic, sequenceQueueMaxLen(q.maxLen, data))
	if err := store.ensureMetadata(ctx); err != nil {
		return err
	}
	result, err := store.enqueue(ctx, customerID, data)
	if err != nil {
		return err
	}
	switch result.Code {
	case sequenceQueueCodeScheduled, sequenceQueueCodeQueued:
		return nil
	case sequenceQueueCodeFull:
		return fmt.Errorf("sequence queue pending capacity reached: %d", result.Pending)
	default:
		return fmt.Errorf("sequence queue enqueue rejected: %s", result.Code)
	}
}

func (q *SequenceQueue) ConsumerSequence(ctx context.Context, channel, topic string, do public.CallbackWithRetry) {
	store := q.sequenceQueueStore(channel, topic, q.maxLen)
	runtime := newSequenceQueueRuntime(store, q.base.processLogger, q.base.consumerPoolSize,
		func(ctx context.Context, channel, topic, raw string, handler public.CallbackWithRetry) (map[string]any, error) {
			return executeSequenceQueueMessage(ctx, channel, topic, raw, handler, q.base.captureConfig)
		})
	runtime.run(ctx, channel, topic, do)
}

func (q *SequenceQueue) sequenceQueueStore(channel, topic string, maxLen int64) *sequenceQueueStore {
	topology := newSequenceQueueTopology(q.base.prefix, channel, topic, q.partitions)
	return newSequenceQueueStore(q.base.client, topology, maxLen, q.base.deadLetterIdle)
}

func sequenceQueueMaxLen(defaultMaxLen int64, data map[string]any) int64 {
	if maxLen := cast.ToInt64(data["maxLen"]); maxLen > 0 {
		return maxLen
	}
	return defaultMaxLen
}

func sequenceQueueFailedData(channel, topic, customerID, raw string, cause error) map[string]any {
	data := make(map[string]any)
	if err := bjson.Unmarshal([]byte(raw), &data); err != nil {
		data["payload"] = raw
	}
	now := time.Now()
	data["channel"] = channel
	data["topic"] = topic
	data["customerId"] = customerID
	data["moodType"] = btype.SEQUENCE_QUEUE
	data["status"] = bstatus.StatusFailed
	data["level"] = bstatus.ErrLevel
	data["info"] = cause.Error()
	data["beginTime"] = now
	data["endTime"] = now
	data["runTime"] = 0
	return data
}

func sequenceQueuePartitionName(partition int64) string {
	return fmt.Sprintf("stream%03d", partition)
}

func sequenceQueueMessageRoute(data map[string]any) (channel, topic, customerID string) {
	channel = cast.ToString(data["channel"])
	topic = cast.ToString(data["topic"])
	customerID = cast.ToString(data["customerId"])
	return channel, topic, customerID
}
