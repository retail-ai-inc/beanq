package main

import (
	"context"
	"fmt"

	"github.com/retail-ai-inc/beanq/v4"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
)

func main() {
	config, err := beanq.NewConfig("./", "json", "env")
	if err != nil {
		logger.New().Error(err)
		return
	}
	csm := beanq.New(config)
	_, _ = csm.BQ().SubscribeToSequenceByLock("delay-channel", "order-topic", beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, message *beanq.Message) error {
			fmt.Printf("---%+v \n", message)
			return nil
		},
		DoCancel: nil,
		DoError:  nil,
	})

	ctx := context.Background()

	// begin to consume information
	csm.Wait(ctx)
}
