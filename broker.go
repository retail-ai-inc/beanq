package beanq

import (
	"context"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/logger"
)

type (
	IBroker interface {
		getMessageInQueue(ctx context.Context, channel, topic string, id string) (*Message, error)
		checkStatus(ctx context.Context, channel, topic string, id string) (string, error)
		enqueue(ctx context.Context, msg *Message, dynamicOn bool) error
		startConsuming(ctx context.Context)
		addConsumer(subscribeType subscribeType, channel, topic string, subscribe IConsumeHandle) *RedisHandle
		addDynamicConsumer(subType subscribeType, channel, topic string, subscribe IConsumeHandle, streamKey, dynamicKey string) *RedisHandle
		deadLetter(ctx context.Context, handle IHandle) error
		dynamicConsuming(subType subscribeType, channel string, subscribe IConsumeHandle, dynamicKey string)
	}
)

var (
	broker     IBroker
	brokerOnce sync.Once
)

// NewBroker ...
func NewBroker(config *BeanqConfig) IBroker {

	brokerOnce.Do(
		func() {
			pool, err := ants.NewPool(config.ConsumerPoolSize, ants.WithPreAlloc(true), ants.WithNonblocking(true))
			if err != nil {
				logger.New().With("", err).Panic("goroutine pool error")
			}

			switch config.Broker {
			case "redis":
				broker = newRedisBroker(config, pool)
			default:
				logger.New().Panic("not support broker type:", config.Broker)
			}
		},
	)

	return broker
}

// consumer...

type (
	IConsumeHandle interface {
		Handle(ctx context.Context, message *Message) error
	}

	IConsumeCancel interface {
		Cancel(ctx context.Context, message *Message) error
	}

	IConsumeError interface {
		Error(ctx context.Context, err error)
	}
)
type (
	DefaultHandle struct {
		DoHandle func(ctx context.Context, message *Message) error
		DoCancel func(ctx context.Context, message *Message) error
		DoError  func(ctx context.Context, err error)
	}
	WorkflowHandler func(ctx context.Context, wf *Workflow) error
)

func (c WorkflowHandler) Handle(ctx context.Context, message *Message) error {
	workflow := NewWorkflow(message)

	return c(ctx, workflow)
}

func (c DefaultHandle) Handle(ctx context.Context, message *Message) error {
	if c.DoHandle != nil {
		return c.DoHandle(ctx, message)
	}
	return nil
}

func (c DefaultHandle) Cancel(ctx context.Context, message *Message) error {
	if c.DoCancel != nil {
		return c.DoCancel(ctx, message)
	}
	return nil
}

func (c DefaultHandle) Error(ctx context.Context, err error) {
	if c.DoError != nil {
		c.DoError(ctx, err)
	}
}
