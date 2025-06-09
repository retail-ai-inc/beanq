package main

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	beanq "github.com/retail-ai-inc/beanq/v3"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
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

	for i := 0; i < 3; i++ {
		id := cast.ToString(i)

		m := make(map[string]any)
		m["delayMsg"] = "new msg" + id

		b, _ := json.Marshal(m)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		result, err := pub.BQ().WithContext(ctx).
			SetId(id).
			PublishInSequenceByLock("delay-channel", "order-topic", "aa", b).WaitingAck()
		if err != nil {
			logger.New().Error(err, m)
		} else {
			log.Printf("ID:%+v \n", result.Id)
		}
	}

	//Force delete a key
	//pub.ForceUnlock(context.Background(), "delay-channel", "order-topic", "aa")

	// this is a single check for ACK
	// result, berr := pub.CheckAckStatus(context.Background(), "delay-channel", "cp0smosf6ntt0aqcpgtg")
	// if berr != nil {
	// 	panic(berr)
	// }
	// log.Println(result)
}
