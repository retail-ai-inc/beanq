package bredis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/boptions"
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

type BrokerOptions struct {
	Prefix                  string
	MaxLen                  int64
	NormalQueuePartitions   int64
	SequenceQueuePartitions int64
	ConsumerWorkers         int
	ConsumerReaders         int
	DeadLetterIdle          time.Duration
	GracefulShutdownTimeout time.Duration
	ReplicationWait         ReplicationWaitOptions
}

type ReplicationWaitOptions struct {
	Replicas int
	Timeout  time.Duration
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

// NewBroker constructs a broker using one partition per configured consumer.
// Deprecated: use NewBrokerWithOptions.
func NewBroker(client redis.UniversalClient, prefix string, maxLen, consumers int64, consumerPoolSize int, duration time.Duration) *Broker {
	return NewBrokerWithOptions(client, BrokerOptions{
		Prefix: prefix, MaxLen: maxLen,
		NormalQueuePartitions: consumers, SequenceQueuePartitions: consumers,
		ConsumerWorkers: consumerPoolSize, DeadLetterIdle: duration,
	})
}

// NewBrokerWithSequenceQueuePartitions constructs a broker with a fixed
// sequence-queue partition count. The topology is immutable after construction.
// Deprecated: use NewBrokerWithOptions.
func NewBrokerWithSequenceQueuePartitions(client redis.UniversalClient, prefix string, maxLen, consumers, sequenceQueuePartitions int64, consumerPoolSize int, duration time.Duration) *Broker {
	return NewBrokerWithOptions(client, BrokerOptions{
		Prefix: prefix, MaxLen: maxLen,
		NormalQueuePartitions: consumers, SequenceQueuePartitions: sequenceQueuePartitions,
		ConsumerWorkers: consumerPoolSize, DeadLetterIdle: duration,
	})
}

// NewBrokerWithPartitions constructs a broker with explicit queue partition counts.
// Deprecated: use NewBrokerWithOptions.
func NewBrokerWithPartitions(client redis.UniversalClient, prefix string, maxLen, consumers, normalQueuePartitions, sequenceQueuePartitions int64, consumerPoolSize int, duration time.Duration) *Broker {
	return NewBrokerWithOptions(client, BrokerOptions{
		Prefix: prefix, MaxLen: maxLen,
		NormalQueuePartitions: normalQueuePartitions, SequenceQueuePartitions: sequenceQueuePartitions,
		ConsumerWorkers: consumerPoolSize, DeadLetterIdle: duration,
	})
}

// NewBrokerWithReplicationWait constructs a broker with Redis WAIT settings.
// Deprecated: use NewBrokerWithOptions.
func NewBrokerWithReplicationWait(client redis.UniversalClient, prefix string, maxLen, consumers, normalQueuePartitions, sequenceQueuePartitions int64, consumerPoolSize int, duration time.Duration, waitReplicas int, waitTimeout time.Duration) *Broker {
	return NewBrokerWithOptions(client, BrokerOptions{
		Prefix: prefix, MaxLen: maxLen,
		NormalQueuePartitions: normalQueuePartitions, SequenceQueuePartitions: sequenceQueuePartitions,
		ConsumerWorkers: consumerPoolSize, ConsumerReaders: boptions.DefaultOptions.ConsumerReaderPoolSize,
		DeadLetterIdle: duration, ReplicationWait: ReplicationWaitOptions{Replicas: waitReplicas, Timeout: waitTimeout},
	})
}

// NewBrokerWithRuntimePoolsAndReplicationWait constructs a broker with independently
// configurable message worker and partition reader pools.
// Deprecated: use NewBrokerWithOptions.
func NewBrokerWithRuntimePoolsAndReplicationWait(client redis.UniversalClient, prefix string, maxLen, consumers, normalQueuePartitions, sequenceQueuePartitions int64, consumerPoolSize, consumerReaderPoolSize int, duration time.Duration, waitReplicas int, waitTimeout time.Duration) *Broker {
	return NewBrokerWithOptions(client, BrokerOptions{
		Prefix: prefix, MaxLen: maxLen,
		NormalQueuePartitions: normalQueuePartitions, SequenceQueuePartitions: sequenceQueuePartitions,
		ConsumerWorkers: consumerPoolSize, ConsumerReaders: consumerReaderPoolSize,
		DeadLetterIdle: duration, ReplicationWait: ReplicationWaitOptions{Replicas: waitReplicas, Timeout: waitTimeout},
	})
}

// NewBrokerWithOptions is the canonical broker constructor.
func NewBrokerWithOptions(client redis.UniversalClient, options BrokerOptions) *Broker {
	if options.ConsumerReaders == 0 {
		options.ConsumerReaders = boptions.DefaultOptions.ConsumerReaderPoolSize
	}
	wait := replicationWait{replicas: options.ReplicationWait.Replicas, timeout: options.ReplicationWait.Timeout}
	queue := func(partitions int64, config *capture.Config) queueOptions {
		return queueOptions{
			client: client, prefix: options.Prefix, maxLen: options.MaxLen, partitions: partitions,
			runtime:        queueRuntimeOptions{workers: options.ConsumerWorkers, readers: options.ConsumerReaders},
			deadLetterIdle: options.DeadLetterIdle, gracefulShutdownTimeout: options.GracefulShutdownTimeout,
			captureConfig: config, wait: wait,
		}
	}
	normal := func(config *capture.Config) *Normal {
		return newNormalWithOptions(queue(options.NormalQueuePartitions, config))
	}
	delay := func(config *capture.Config) *Schedule {
		return newScheduleWithOptions(queue(options.NormalQueuePartitions, config))
	}
	sequenceQueue := func(config *capture.Config) *SequenceQueue {
		return newSequenceQueueWithOptions(queue(options.SequenceQueuePartitions, config))
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
		prefix:        options.Prefix,
		routes:        routes,
		publishLogger: NewProcessLogWithPartitions(client, options.Prefix, options.NormalQueuePartitions, options.SequenceQueuePartitions),
		status:        NewStatusWithPartitions(client, options.Prefix, options.NormalQueuePartitions, options.SequenceQueuePartitions),
		admin:         NewUITool(client, options.Prefix),
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

func (t *Broker) Close() error {
	if t == nil || t.client == nil {
		return nil
	}
	return t.client.Close()
}

type processLogger interface {
	AddLog(ctx context.Context, data map[string]any) error
}
