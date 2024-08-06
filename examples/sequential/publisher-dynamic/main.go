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
	//
	// wait := sync.WaitGroup{}
	// wait.Add(300)
	// for i := 1; i <= 300; i++ {
	// 	go func(ni int) {
	// 		defer wait.Done()
	//
	// 		data := make(map[string]any)
	// 		data["deviceUuid"] = "device_1"
	// 		data["uuid"] = "test-" + cast.ToString(ni)
	// 		data["amount"] = 700
	// 		data["transactionType"] = 12
	// 		data["cardId"] = "5732542140"
	// 		data["retailerStoreId"] = 1
	// 		data["retailerTerminalId"] = 111
	// 		data["retailerCompanyId"] = 1
	// 		now := time.Now()
	// 		client := resty.New()
	// 		resp, err := client.R().
	// 			SetHeader("Content-Type", "application/json").
	// 			SetBody(data).
	// 			Post("http://127.0.0.1:8888/v1/prepaid/card/deposit")
	// 		if err != nil {
	// 			fmt.Printf("错误:%+v \n", err)
	// 			return
	// 		}
	// 		fmt.Printf("返回值 ：%+v,耗时：%+v \n", string(resp.Body()), time.Now().Sub(now))
	// 	}(i)
	// }
	// wait.Wait()
	// return

	pub := beanq.New(initCnf())
	for i := 0; i < 1000; i++ {
		m := make(map[string]any)
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		r, err := pub.BQ().WithContext(ctx).SetId(cast.ToString(i)).PublishInSequential("default-delay-channel", "mynewstream", b).WaitingAck(ctx, cast.ToString(i))
		if err != nil {
			logger.New().Error(err)
		}
		fmt.Printf("-----msg:%+v \n", r)
	}
	return
	// config := initCnf()
	//
	// for i := 1; i < 800; i++ {
	// 	go func() {
	// 		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	// 		defer cancel()
	// 		m := make(map[string]any, 0)
	// 		m["delayMsg"] = "topic2 new msg" + cast.ToString(i)
	// 		m["Id"] = cast.ToString(i)
	// 		b, _ := json.Marshal(m)
	// 		now := time.Now()
	//
	// 		pub := beanq.New(config)
	// 		result, err := pub.BQ().WithContext(ctx).SetId(cast.ToString(i)).PublishInSequential("default-delay-channel", "mynewstream", b).WaitingPubAck()
	// 		if err != nil {
	// 			logger.New().Error(err)
	// 		} else {
	// 			logger.New().Info(result)
	// 		}
	// 		fmt.Printf("sub:%+v \n", time.Now().Sub(now))
	// 	}()
	// }
	// select {}

	// this is a single check for ACK
	// result, err := pub.CheckAckStatus(context.Background(), "delay-channel", "order-topic", "cp0smosf6ntt0aqcpgtg")
	// if err != nil {
	// 	panic(err)
	// }
	// log.Println(result)
}
