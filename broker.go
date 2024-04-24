package beanq

import (
	"context"
	"errors"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/logger"
)

type (
	Broker interface {
		enqueue(ctx context.Context, msg *Message, options Option) error
		close() error
		startConsuming(ctx context.Context)
		addConsumer(subscribeType subscribeType, channel, topic string, run IConsumeHandle)
	}
)

var (
	broker     *RedisBroker
	brokerOnce sync.Once
)

// NewBroker ...
func NewBroker(config BeanqConfig) Broker {
	brokerOnce.Do(
		func() {
			pool, err := ants.NewPool(config.ConsumerPoolSize, ants.WithPreAlloc(true))
			if err != nil {
				logger.New().With("", err).Panic("goroutine pool error")
			}

			switch config.Broker {
			case "redis":
				broker = newRedisBroker(pool)
			default:
				logger.New().With("", err).Panic("not support broker type:", config.Broker)
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
)

func (c DefaultHandle) Handle(ctx context.Context, message *Message) error {
	if c.DoHandle != nil {
		return c.DoHandle(ctx, message)
	}
	return errors.New("missing handle function")
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
