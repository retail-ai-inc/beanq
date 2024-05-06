package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/retail-ai-inc/beanq"
	"github.com/spf13/viper"
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
	client := beanq.New(&config)
	ctx := context.Background()
	// client.Channel("").Topic("").Payload([]byte("aaaaa")).PublishAtTime(ctx, time.Now().Add(10*time.Second))

	client.Channel("").Topic("").Subscribe(ctx, beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, message *beanq.Message) error {
			fmt.Println(message)
			return nil
		},
		DoCancel: nil,
		DoError:  nil,
	})
	client.Wait(ctx)

}
