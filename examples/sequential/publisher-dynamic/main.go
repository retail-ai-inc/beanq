package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
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

	for i := 0; i < 200; i++ {
		m := make(map[string]any)
		m["deviceUuid"] = "device_1"
		m["uuid"] = "fd768e9b" + cast.ToString(i)
		m["amount"] = 700
		m["transactionType"] = 12
		m["cardId"] = "5732542140"
		m["retailerStoreId"] = 1
		m["retailerTerminalId"] = 111
		m["retailerCompanyId"] = 1
		client := resty.New()
		now := time.Now()
		// POST JSON string
		// No need to set content type, if you have client level setting
		res, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(m).
			Post("http://127.0.0.1:8888/v1/prepaid/card/deposit")
		if err != nil {

		}
		fmt.Printf("%+v \n", string(res.Body()))
		fmt.Printf("sub:%+v \n", time.Now().Sub(now))
	}

	return
	// for i := 0; i < 1000; i++ {
	// 	m["delayMsg"] = "new msg" + cast.ToString(i)
	// 	b, _ := json.Marshal(m)
	//
	// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	// 	defer cancel()
	//
	// 	_, err := pub.BQ().WithContext(ctx).Dynamic().PublishInSequential("delay-channel", "order-topic-"+strconv.Itoa(i%100), b).WaitingAck()
	// 	if err != nil {
	// 		logger.New().Error(err)
	// 	}
	// }
	config := initCnf()

	for i := 1; i < 800; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			m := make(map[string]any, 0)
			m["delayMsg"] = "topic2 new msg" + cast.ToString(i)
			m["Id"] = cast.ToString(i)
			b, _ := json.Marshal(m)
			now := time.Now()

			pub := beanq.New(config)
			result, err := pub.BQ().WithContext(ctx).SetId(cast.ToString(i)).PublishInSequential("default-delay-channel", "mynewstream", b).WaitingPubAck()
			if err != nil {
				logger.New().Error(err)
			} else {
				logger.New().Info(result)
			}
			fmt.Printf("sub:%+v \n", time.Now().Sub(now))
		}()
	}
	select {}

	// this is a single check for ACK
	// result, err := pub.CheckAckStatus(context.Background(), "delay-channel", "order-topic", "cp0smosf6ntt0aqcpgtg")
	// if err != nil {
	// 	panic(err)
	// }
	// log.Println(result)
}
