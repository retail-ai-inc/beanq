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
		addConsumer(subscribeType subscribeType, channel, topic string, run ConsumerFunc)
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
	ConsumerCallbackType uint8
	ConsumerFunc         map[ConsumerCallbackType]func(ctx context.Context, data any) error
)

const (
	ConsumerHandle ConsumerCallbackType = iota + 1
	ConsumerError
	ConsumerCancel
)

func (c ConsumerFunc) Handle(ctx context.Context, message *Message) error {
	if h, ok := c[ConsumerHandle]; ok {
		return h(ctx, message)
	}
	return errors.New("missing handle function")
}

func (c ConsumerFunc) Cancel(ctx context.Context, message *Message) error {
	if h, ok := c[ConsumerCancel]; ok {
		return h(ctx, message)
	}
	return nil
}

func (c ConsumerFunc) Error(ctx context.Context, err error) error {
	if h, ok := c[ConsumerError]; ok {
		return h(ctx, err)
	}
	return nil
}
