package main

import (
	"context"
	"log"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	beanq "github.com/retail-ai-inc/beanq/v4"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
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
	csm := beanq.New(config)

	// register delay consumer
	ctx := context.Background()
	_, err := csm.BQ().WithContext(ctx).Subscribe("default-channel", "default-topic", beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, message *beanq.Message) error {
			time.Sleep(20 * time.Second)
			logger.New().With("default-channel", "default-topic").Info(message.Payload)
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
	// begin to consume information
	csm.Wait(ctx)
}
