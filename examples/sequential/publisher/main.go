package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq/v4"
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

	now := time.Now()
	fmt.Printf("now:%+v \n", now)
	wg := sync.WaitGroup{}

	n := 10
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go func() {
			defer wg.Done()
			m := make(map[string]any, 2)
			m["delayMsg"] = "new msg" + cast.ToString(i)
			m["id"] = cast.ToString(i)
			b, _ := json.Marshal(m)
			bq := pub.BQ()
			ctx := context.Background()
			if err := bq.WithContext(ctx).SetId(cast.ToString(i)).PublishInSequence("sequential-channel", "order-topic", b).Error(); err != nil {
				logger.New().Error(err)
			}
		}()
	}

	wg.Wait()
	fmt.Printf("after:%+v,sub:%+v \n", time.Now(), time.Since(now))
}
