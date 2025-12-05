package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq/v4"
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

		envPath := "./"
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
	pubMoreAndPriorityInfo()
}

func pubMoreAndPriorityInfo() {
	pub := beanq.New(initCnf())
	m := make(map[string]string)

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		m["delayMsg"] = "testing --------" + cast.ToString(i)
		b, _ := json.Marshal(m)

		if err := pub.BQ().WithContext(ctx).SetTimeToRun(20*time.Second, 10*time.Second, 15*time.Second).Publish("default-channel", "default-topic", b); err != nil {
			logger.New().Error(err)
		}
	}
}
