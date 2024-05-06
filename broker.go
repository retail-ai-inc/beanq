package beanq

import (
	"context"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/logger"
)

type (
	IBroker interface {
		enqueue(ctx context.Context, msg *Message) error
		close() error
		startConsuming(ctx context.Context)
		addConsumer(subscribeType subscribeType, channel, topic string, subscribe IConsumeHandle)
		deadLetter(ctx context.Context, handle IHandle) error
		check(ctx context.Context, subType subscribeType, channel, topic string) error
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
			pool, err := ants.NewPool(config.ConsumerPoolSize, ants.WithPreAlloc(true))
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
)

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
