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
	base       queueBase
	maxLen     int64
	partitions int64
	wait       replicationWait
}

func NewSequenceQueue(client redis.UniversalClient, prefix string, maxLen int64, consumerCount int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config) *SequenceQueue {
	return newSequenceQueueWithPartitions(client, prefix, maxLen, consumerCount, consumerPoolSize, deadLetterIdle, config)
}

func newSequenceQueueWithPartitions(client redis.UniversalClient, prefix string, maxLen, partitions int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config) *SequenceQueue {
	return newSequenceQueueWithPartitionsAndWait(client, prefix, maxLen, partitions, consumerPoolSize, deadLetterIdle, config, replicationWait{})
}

func newSequenceQueueWithPartitionsAndWait(client redis.UniversalClient, prefix string, maxLen, partitions int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config, wait replicationWait) *SequenceQueue {
	partitions = normalizeSequenceQueuePartitionCount(partitions)
	base := newQueueBase(queueBaseOptions{client: client, prefix: prefix,
		deadLetterIdle: deadLetterIdle, consumerPoolSize: consumerPoolSize, captureConfig: config, wait: wait})
	base.processLogger = NewProcessLog(client, prefix, partitions)
	return &SequenceQueue{
		maxLen:     maxLen,
		partitions: partitions,
		wait:       wait,
		base:       base,
	}
}

func (q *SequenceQueue) PublishNewSequence(ctx context.Context, data map[string]any) error {
	channel, topic, orderKey := sequenceQueueMessageRoute(data)
	if orderKey == "" {
		return errors.New("missing orderKey")
	}
	store := q.sequenceQueueStore(channel, topic, q.maxLen)
	if err := store.ensureMetadata(ctx); err != nil {
		return err
	}
	result, err := store.enqueueWithWait(ctx, orderKey, data, q.wait)
	if err != nil {
		return err
	}
	switch result.Code {
	case bstatus.SequenceQueueCodeScheduled, bstatus.SequenceQueueCodeQueued:
		return nil
	case bstatus.SequenceQueueCodeFull:
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
	return newSequenceQueueStore(q.base.client, topology, maxLen, q.base.deadLetterIdle, q.wait)
}

func sequenceQueueFailedData(channel, topic, orderKey, raw string, cause error) map[string]any {
	data := make(map[string]any)
	if err := bjson.Unmarshal([]byte(raw), &data); err != nil {
		data["payload"] = raw
	}
	now := time.Now()
	data["channel"] = channel
	data["topic"] = topic
	data["orderKey"] = orderKey
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

func sequenceQueueMessageRoute(data map[string]any) (channel, topic, orderKey string) {
	channel = cast.ToString(data["channel"])
	topic = cast.ToString(data["topic"])
	orderKey = cast.ToString(data["orderKey"])
	return channel, topic, orderKey
}
