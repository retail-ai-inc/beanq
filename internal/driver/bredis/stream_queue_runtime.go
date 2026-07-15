package bredis

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	public "github.com/retail-ai-inc/beanq/v4/internal"
)

const streamQueueReadBatch = int64(16)

type streamQueueDispatch struct {
	stream  string
	message redis.XMessage
}

type streamQueueAdapter struct {
	name               string
	consumerPrefix     string
	instanceID         string
	client             redis.UniversalClient
	base               *queueBase
	partitions         int64
	ensureMetadata     func(context.Context) error
	bootstrapGroups    func(context.Context, string) error
	bootstrapPartition func(context.Context, string, int64) error
	streamKey          func(int64) string
	deadLetterLockKey  func(int64) string
	startExtra         func(context.Context)
}

func (a *streamQueueAdapter) Name() string                             { return a.name }
func (a *streamQueueAdapter) ConsumerPrefix() string                   { return a.consumerPrefix }
func (a *streamQueueAdapter) InstanceID() string                       { return a.instanceID }
func (a *streamQueueAdapter) Partitions() int64                        { return a.partitions }
func (a *streamQueueAdapter) Workers() int                             { return a.base.consumerPoolSize }
func (a *streamQueueAdapter) DispatchCapacity() int                    { return max(1, a.Workers()) * 2 }
func (a *streamQueueAdapter) EnsureMetadata(ctx context.Context) error { return a.ensureMetadata(ctx) }
func (a *streamQueueAdapter) BootstrapGroups(ctx context.Context, group string) error {
	return a.bootstrapGroups(ctx, group)
}
func (a *streamQueueAdapter) BootstrapPartition(ctx context.Context, group string, partition int64) error {
	return a.bootstrapPartition(ctx, group, partition)
}
func (a *streamQueueAdapter) StartBackground(ctx context.Context, channel, topic string) {
	for partition := int64(0); partition < a.partitions; partition++ {
		partition := partition
		go a.base.DeadLetterStream(ctx, channel, topic, a.streamKey(partition), a.deadLetterLockKey(partition))
	}
	if a.startExtra != nil {
		go a.startExtra(ctx)
	}
}
func (a *streamQueueAdapter) Claim(ctx context.Context, group, consumer string, partition int64, _ string) ([]streamQueueDispatch, string, error) {
	stream := a.streamKey(partition)
	messages, _, err := a.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream: stream, Group: group, Consumer: consumer, MinIdle: a.base.deadLetterIdle, Start: "0-0", Count: streamQueueReadBatch,
	}).Result()
	items := make([]streamQueueDispatch, 0, len(messages))
	for _, message := range messages {
		items = append(items, streamQueueDispatch{stream: stream, message: message})
	}
	return items, "", err
}
func (a *streamQueueAdapter) Read(ctx context.Context, group, consumer string, partition int64) ([]streamQueueDispatch, error) {
	stream := a.streamKey(partition)
	streams, err := a.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: group, Consumer: consumer, Streams: []string{stream, ">"}, Count: streamQueueReadBatch, Block: partitionNonBlockingRead,
	}).Result()
	if err != nil {
		return nil, err
	}
	var items []streamQueueDispatch
	for _, result := range streams {
		for _, message := range result.Messages {
			items = append(items, streamQueueDispatch{stream: result.Stream, message: message})
		}
	}
	return items, nil
}
func (a *streamQueueAdapter) Process(ctx context.Context, _, _, group, _ string, item streamQueueDispatch, handler public.CallbackWithRetry) {
	result, ok := executeMessage(ctx, public.Stream{Data: item.message.Values, Id: item.message.ID, Channel: group, Stream: item.stream}, handler, a.base.captureConfig)
	if !ok {
		return
	}
	if err := a.base.addLog(ctx, result.Data); err != nil {
		logger.New().Error(err)
		return
	}
	if err := ackAndDelete(ctx, a.client, item.stream, group, item.message.ID); err != nil {
		logger.New().Error(err)
	}
}
