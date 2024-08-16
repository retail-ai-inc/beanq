package main

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

var (
	configOnce sync.Once
	bqConfig   beanq.BeanqConfig
)

func initCnf() *beanq.BeanqConfig {
	configOnce.Do(func() {
		var envPath string = "./"
		if _, file, _, ok := runtime.Caller(0); ok {
			envPath = filepath.Dir(file)
		}

		vp := viper.New()
		vp.AddConfigPath(envPath)
		vp.SetConfigType("json")
		vp.SetConfigName("env")

		if err := vp.ReadInConfig(); err != nil {
			log.Fatalf("Unable to open beanq env.json file: %v", err)
		}

		// IMPORTANT: Unmarshal the env.json into global Config object.
		if err := vp.Unmarshal(&bqConfig); err != nil {
			log.Fatalf("Unable to unmarshal the beanq env.json file: %v", err)
		}
	})
	return &bqConfig
}
func main() {

	config := initCnf()
	pub := beanq.New(config)
	wait := sync.WaitGroup{}

	for i := 0; i < 5; i++ {
		wait.Add(1)
		go func(i1 int) {
			defer wait.Done()
			id := cast.ToString(i1)

			m := make(map[string]any)
			m["delayMsg"] = "new msg" + id

			b, _ := json.Marshal(m)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()
			result, err := pub.BQ().WithContext(ctx).SetId(id).PublishInSequential("delay-channel", "order-topic", b).WaitingAck()
			if err != nil {
				logger.New().Error(err)
			} else {
				log.Printf("%+v \n", result)
			}
		}(i)

	}
	wait.Wait()
	// this is a single check for ACK
	// result, err := pub.CheckAckStatus(context.Background(), "delay-channel", "cp0smosf6ntt0aqcpgtg")
	// if err != nil {
	// 	panic(err)
	// }
	// log.Println(result)
}
