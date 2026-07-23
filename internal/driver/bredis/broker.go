package bredis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/spf13/cast"
)

type Broker struct {
	client        redis.UniversalClient
	prefix        string
	routes        map[btype.MoodType]queueRoute
	publishLogger processLogger
	status        *Status
	admin         *UITool
	migrator      *Log
	captureConfig *capture.Config
}

type Consumer interface {
	Consume(ctx context.Context, channel, topic string, handler public.CallbackWithRetry)
}

type consumeFunc func(ctx context.Context, channel, topic string, handler public.CallbackWithRetry)

func (f consumeFunc) Consume(ctx context.Context, channel, topic string, handler public.CallbackWithRetry) {
	f(ctx, channel, topic, handler)
}

type queueRoute struct {
	publish func(ctx context.Context, data map[string]any) error
	consume func(config *capture.Config) Consumer
}

func NewBroker(client redis.UniversalClient, prefix string, maxLen, consumers int64, consumerPoolSize int, duration time.Duration) *Broker {
	return NewBrokerWithPartitions(client, prefix, maxLen, consumers, consumers, consumers, consumerPoolSize, duration)
}

// NewBrokerWithSequenceQueuePartitions constructs a broker with a fixed
// sequence-queue partition count. The topology is immutable after construction.
func NewBrokerWithSequenceQueuePartitions(client redis.UniversalClient, prefix string, maxLen, consumers, sequenceQueuePartitions int64, consumerPoolSize int, duration time.Duration) *Broker {
	return NewBrokerWithPartitions(client, prefix, maxLen, consumers, consumers, sequenceQueuePartitions, consumerPoolSize, duration)
}

func NewBrokerWithPartitions(client redis.UniversalClient, prefix string, maxLen, consumers, normalQueuePartitions, sequenceQueuePartitions int64, consumerPoolSize int, duration time.Duration) *Broker {
	return NewBrokerWithReplicationWait(client, prefix, maxLen, consumers, normalQueuePartitions, sequenceQueuePartitions, consumerPoolSize, duration, 0, 0)
}

func NewBrokerWithReplicationWait(client redis.UniversalClient, prefix string, maxLen, consumers, normalQueuePartitions, sequenceQueuePartitions int64, consumerPoolSize int, duration time.Duration, waitReplicas int, waitTimeout time.Duration) *Broker {
	wait := replicationWait{replicas: waitReplicas, timeout: waitTimeout}
	normal := func(config *capture.Config) *Normal {
		return newNormalWithPartitionsAndWait(client, prefix, maxLen, normalQueuePartitions, consumerPoolSize, duration, config, wait)
	}
	delay := func(config *capture.Config) *Schedule {
		return newScheduleWithPartitionsAndWait(client, prefix, maxLen, normalQueuePartitions, consumerPoolSize, duration, config, wait)
	}
	sequenceQueue := func(config *capture.Config) *SequenceQueue {
		return newSequenceQueueWithPartitionsAndWait(client, prefix, maxLen, sequenceQueuePartitions, consumerPoolSize, duration, config, wait)
	}
	routes := map[btype.MoodType]queueRoute{
		btype.NORMAL: routeFromQueue(normal(nil).Publish, func(config *capture.Config) consumeFunc { return normal(config).Consume }),
		btype.DELAY:  routeFromQueue(delay(nil).Publish, func(config *capture.Config) consumeFunc { return delay(config).Consume }),
		btype.SEQUENCE_QUEUE: routeFromQueue(sequenceQueue(nil).PublishNewSequence, func(config *capture.Config) consumeFunc {
			return sequenceQueue(config).ConsumerSequence
		}),
	}
	return &Broker{
		client:        client,
		prefix:        prefix,
		routes:        routes,
		publishLogger: NewProcessLog(client, prefix, sequenceQueuePartitions),
		status:        NewStatus(client, prefix, sequenceQueuePartitions),
		admin:         NewUITool(client, prefix),
	}
}

func routeFromQueue(publish func(context.Context, map[string]any) error, consume func(*capture.Config) consumeFunc) queueRoute {
	return queueRoute{
		publish: publish,
		consume: func(config *capture.Config) Consumer { return consume(config) },
	}
}

func (t *Broker) Enqueue(ctx context.Context, data map[string]any) error {
	moodType := btype.NORMAL
	if v, ok := data["moodType"]; ok {
		moodType = btype.MoodType(cast.ToString(v))
	}

	route, ok := t.routes[moodType]
	if !ok || route.publish == nil {
		return berror.BrokerDriverError
	}
	if err := route.publish(ctx, data); err != nil {
		return err
	}

	data["status"] = bstatus.StatusPublished
	return t.publishLogger.AddLog(ctx, data)
}

func (t *Broker) Consumer(moodType btype.MoodType, config *capture.Config) (Consumer, error) {
	route, ok := t.routes[moodType]
	if !ok || route.consume == nil {
		return nil, berror.BrokerDriverError
	}
	return route.consume(config), nil
}

func (t *Broker) Configure(captureConfig *capture.Config, migrator migrateStore) {
	t.captureConfig = captureConfig
	t.migrator = NewLog(t.client, t.prefix, migrator)
}

func (t *Broker) Consume(ctx context.Context, moodType btype.MoodType, channel, topic string, callback public.CallbackWithRetry) error {
	consumer, err := t.Consumer(moodType, t.captureConfig)
	if err != nil {
		return err
	}
	consumer.Consume(ctx, channel, topic, callback)
	return nil
}

func (t *Broker) WaitingAck(ctx context.Context, channel, topic, id string) (map[string]string, error) {
	return t.status.Status(ctx, channel, topic, id)
}

func (t *Broker) WaitingSequenceAck(ctx context.Context, channel, topic, orderKey, id string) (map[string]string, error) {
	return t.status.SequenceStatus(ctx, channel, topic, orderKey, id)
}

func (t *Broker) Migrate(ctx context.Context, data []map[string]any) error {
	if t.migrator == nil {
		return nil
	}
	return t.migrator.Migrate(ctx, data)
}

func (t *Broker) HostName(ctx context.Context) error {
	return t.admin.HostName(ctx)
}

func (t *Broker) QueueMessage(ctx context.Context) error {
	return t.admin.QueueMessage(ctx)
}

func (t *Broker) Driver() any {
	return t.client
}

type processLogger interface {
	AddLog(ctx context.Context, data map[string]any) error
}
