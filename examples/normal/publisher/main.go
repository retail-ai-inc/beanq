package main

import (
	"context"

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
	pub := beanq.New(config)

	b := []byte(`{"msg":"publish testing"}`)
	if err := pub.BQ().WithContext(context.Background()).Publish(channel, topic, b); err != nil {
		logger.New().Error(err)
	}

}
