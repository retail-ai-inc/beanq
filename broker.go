package beanq

import (
	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/logger"
)

func NewBroker(config BeanqConfig) Broker {
	pool, err := ants.NewPool(config.ConsumerPoolSize, ants.WithPreAlloc(true))
	if err != nil {
		logger.New().With("", err).Fatal("goroutine pool error")
	}

	switch config.Broker {
	case "redis":
		return newRedisBroker(pool)
	default:
		logger.New().With("", err).Panic("not support broker type:", config.Broker)
	}

	return nil
}
