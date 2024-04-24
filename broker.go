package beanq

import (
	"context"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	"github.com/retail-ai-inc/beanq/helper/logger"
)

type (
	Broker interface {
		enqueue(ctx context.Context, msg *Message, options Option) error
		close() error
		startConsuming(ctx context.Context)
		addConsumer(subscribeType subscribeType, channel, topic string, run Handler)
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

// consumer ...
type (
	Handler interface {
		Handle(ctx context.Context, message *Message) error
	}
	Cancel interface {
		Cancel(ctx context.Context, message *Message) error
	}
	Error interface {
		Error(ctx context.Context, err error) error
	}
	ConsumerCallback struct {
		handler       func(ctx context.Context, message *Message) error
		cancelHandler func(ctx context.Context, message *Message) error
		errorHandler  func(ctx context.Context, err error) error
	}
)

func NewConsumerCallback() *ConsumerCallback {
	return &ConsumerCallback{}
}

func (c *ConsumerCallback) AddHandler(handler func(ctx context.Context, message *Message) error) *ConsumerCallback {
	c.handler = handler
	return c
}

func (c *ConsumerCallback) AddCancelHandler(cancel func(ctx context.Context, message *Message) error) *ConsumerCallback {
	c.cancelHandler = cancel
	return c
}

func (c *ConsumerCallback) AddErrorHandler(error func(ctx context.Context, err error) error) *ConsumerCallback {
	c.errorHandler = error
	return c
}

func (c *ConsumerCallback) Handle(ctx context.Context, message *Message) error {
	if c.handler == nil {
		return errors.New("missing handle function")
	}
	return c.handler(ctx, message)
}

func (c *ConsumerCallback) Cancel(ctx context.Context, message *Message) error {
	if c.cancelHandler == nil {
		return nil
	}
	return c.cancelHandler(ctx, message)
}

func (c *ConsumerCallback) Error(ctx context.Context, err error) error {
	if c.errorHandler == nil {
		return nil
	}
	return c.errorHandler(ctx, err)
}
