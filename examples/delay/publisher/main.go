package main

import (
	"context"
	"time"

	"github.com/retail-ai-inc/beanq/v4"
	"github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/spf13/cast"
)

func main() {
	config, err := beanq.NewConfig("./", "json", "env")
	if err != nil {
		logger.New().Error(err)
		return
	}
	pub := beanq.New(config)

	m := make(map[string]any)
	ctx := context.Background()
	now := time.Now()

	//Sort by execution time, the smaller the priority, the earlier it is consumed.
	//For messages of the same time, the larger the priority, the earlier it is consumed.
	for i := 0; i < 10; i++ {

		delayT := now
		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)

		b, _ := json.Marshal(m)

		if i == 4 {
			//setting priority
			y = 8
		}
		if i == 3 {
			// delay execution time
			delayT = now.Add(10 * time.Second)
		}

		if err := pub.BQ().WithContext(ctx).Priority(float64(y)).PublishAtTime("delay-channel", "order-topic", b, delayT); err != nil {
			logger.New().Error(err)
		}
	}
}
