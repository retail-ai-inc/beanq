package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sync"

	beanq "github.com/retail-ai-inc/beanq/v4"
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
	config := initCnf()
	csm := beanq.New(config)
	_, _ = csm.BQ().SubscribeToSequenceByLock("delay-channel", "order-topic", beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, message *beanq.Message) error {
			fmt.Printf("---%+v \n", message)
			return nil
		},
		DoCancel: nil,
		DoError:  nil,
	})

	ctx := context.Background()

	// begin to consume information
	csm.Wait(ctx)
}
