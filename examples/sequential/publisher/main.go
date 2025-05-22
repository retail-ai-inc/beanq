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
			if err := bq.WithContext(ctx).SetId(cast.ToString(i)).PublishInSequential("delay-channel", "order-topic", b).Error(); err != nil {
				logger.New().Error(err)
			}
		}()
	}

	wg.Wait()
	fmt.Printf("after:%+v,sub:%+v \n", time.Now(), time.Now().Sub(now))
}
