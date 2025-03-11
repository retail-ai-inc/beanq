package main

import (
	"context"
	beanq "github.com/retail-ai-inc/beanq/v3"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/spf13/viper"
	"log"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	configOnce sync.Once
	bqConfig   beanq.BeanqConfig
)

func initCnf() beanq.BeanqConfig {
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
	return bqConfig
}

func main() {
	config := initCnf()
	ctx := context.Background()
	csm := beanq.New(&config)

	// register delay consumer

	_, err := csm.BQ().WithContext(ctx).SubscribeDelay("delay-channel", "order-topic", beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, message *beanq.Message) error {
			logger.New().With("delay-channel", "delay-topic--------").Info(message.Payload)
			return nil
		},
		DoCancel: func(ctx context.Context, message *beanq.Message) error {
			return nil
		},
		DoError: func(ctx context.Context, err error) {
			logger.New().Error(err)
		},
	})
	if err != nil {
		logger.New().Error(err)
	}

	csm.Wait(ctx)

}
