package main

import (
	"context"
	"encoding/json"
	"log"
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
	wait := sync.WaitGroup{}

	for i := 0; i < 300; i++ {
		wait.Add(1)
		go func(i1 int) {
			defer wait.Done()
			id := cast.ToString(i1)

			m := make(map[string]any)
			m["delayMsg"] = "new msg" + id

			b, _ := json.Marshal(m)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()
			result, err := pub.BQ().WithContext(ctx).SetId(id).PublishInSequence("delay-channel", "order-topic", b).WaitingAck()
			if err != nil {
				logger.New().Error(err, m)
			} else {
				log.Printf("ID:%+v \n", result.Id)
			}
		}(i)

	}
	wait.Wait()
	// this is a single check for ACK
	// result, berr := pub.CheckAckStatus(context.Background(), "delay-channel", "cp0smosf6ntt0aqcpgtg")
	// if berr != nil {
	// 	panic(berr)
	// }
	// log.Println(result)
}
