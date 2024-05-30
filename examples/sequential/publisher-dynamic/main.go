package main

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"
	"runtime"
	"strconv"
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

	m := make(map[string]any)
	for i := 0; i < 15; i++ {
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		err := pub.BQ().WithContext(ctx).Dynamic().PublishInSequential("delay-channel", "order-topic-"+strconv.Itoa(i%5), b).Error()
		if err != nil {
			logger.New().Error(err)
		}
	}

	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		m["delayMsg"] = "topic2 new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)
		result, err := pub.BQ().WithContext(ctx).Dynamic().PublishInSequential("delay-channel", "order-topic-n", b).WaitingAck()
		if err != nil {
			logger.New().Error(err)
		} else {
			log.Println(result)
		}
	}

	// this is a single check for ACK
	result, err := pub.CheckAckStatus(context.Background(), "delay-channel", "order-topic", "cp0smosf6ntt0aqcpgtg")
	if err != nil {
		panic(err)
	}
	log.Println(result)
}
