package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	beanq "github.com/retail-ai-inc/beanq/v4"
	"github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
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
	pubDelayInfo()
}

func pubDelayInfo() {
	config := initCnf()
	pub := beanq.New(config)

	m := make(map[string]any)
	ctx := context.Background()
	now := time.Now()
	delayT := now
	for i := 0; i < 10; i++ {

		delayT = now
		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)

		b, _ := json.Marshal(m)

		if i == 2 {
			delayT = now
		}

		if i == 4 {
			y = 8
		}
		if i == 3 {
			y = 10
			delayT = now.Add(10 * time.Second)
		}
		// continue
		if err := pub.BQ().WithContext(ctx).Priority(float64(y)).PublishAtTime("delay-channel", "order-topic", b, delayT); err != nil {
			logger.New().Error(err)
		}
	}
}
