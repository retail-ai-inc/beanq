package beanq

import (
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/logger"
)

var (
	broker     *RedisBroker
	brokerOnce sync.Once
)

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
