package beanq

import (
	"context"
	"errors"
	"sync"

	"github.com/retail-ai-inc/beanq/helper/logger"
)

type (
	IBroker interface {
		checkStatus(ctx context.Context, channel, id string) (*Message, error)
		enqueue(ctx context.Context, msg *Message, dynamicOn bool) error
		startConsuming(ctx context.Context)
		addConsumer(subscribeType subscribeType, channel, topic string, subscribe IConsumeHandle) *RedisHandle
		addDynamicConsumer(subType subscribeType, channel, topic string, subscribe IConsumeHandle, streamKey, dynamicKey string) *RedisHandle
		dynamicConsuming(subType subscribeType, channel string, subscribe IConsumeHandle, dynamicKey string)

		monitorStream(ctx context.Context, channel, id string) (*Message, error)
		setCaptureException(fn func(ctx context.Context, err any))
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
			switch config.Broker {
			case "redis":
				broker = newRedisBroker(config)
			default:
				logger.New().Panic("not support broker type:", config.Broker)
			}
		},
	)

	return broker
}

// consumer...
var (
	NilHandle = errors.New("beanq:handle is nil")
	NilCancel = errors.New("beanq:cancel is nil")
)

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
	return NilHandle
}

func (c DefaultHandle) Cancel(ctx context.Context, message *Message) error {
	if c.DoCancel != nil {
		return c.DoCancel(ctx, message)
	}
	return NilCancel
}

func (c DefaultHandle) Error(ctx context.Context, err error) {
	if c.DoError != nil {
		c.DoError(ctx, err)
	}
}
