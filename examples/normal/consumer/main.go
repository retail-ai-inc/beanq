package main

import (
	"context"
	"time"

	"github.com/retail-ai-inc/beanq/v4"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
)

func main() {

	channel := "default-channel"
	topic := "default-topic"

	config, err := beanq.NewConfig("./", "json", "env")
	if err != nil {
		logger.New().Error(err)
		return
	}
	consumer := beanq.New(config)

	// register delay consumer
	ctx := context.Background()
	_, err = consumer.BQ().WithContext(ctx).Subscribe(channel, topic, beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, message *beanq.Message) error {
			time.Sleep(20 * time.Second)
			logger.New().With("default-channel", "default-topic").Info(message.Payload)
			return nil
		},
		DoCancel: func(ctx context.Context, message *beanq.Message) error {
			return nil
		},
		DoError: func(ctx context.Context, err error) {
			logger.New().Error(err)

		},
	})
	if err != nil {
		logger.New().Error(err)
	}
	// begin to consume information
	consumer.Wait(ctx)
}
