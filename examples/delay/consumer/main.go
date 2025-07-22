package main

import (
	"context"

	"github.com/retail-ai-inc/beanq/v4"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
)

func main() {

	channel := "delay-channel"
	topic := "delay-topic"

	config, err := beanq.NewConfig("./", "json", "env")
	if err != nil {
		logger.New().Error(err)
		return
	}
	ctx := context.Background()
	csm := beanq.New(config)

	// register delay consumer

	_, err = csm.BQ().WithContext(ctx).SubscribeToDelay(channel, topic, beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, message *beanq.Message) error {
			logger.New().With("delay-channel", "delay-topic--------").Info(message.Payload)
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

	csm.Wait(ctx)

}
