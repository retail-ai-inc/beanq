package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/workflow"
	"github.com/rs/xid"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	pipe := workflow.NewPipe()
	pub := beanq.NewPublisher(initCnf())
	pipe.AddCargo(xid.New().String(), func(ctx context.Context, id string) (*beanq.Message, error) {
		msg := beanq.NewMessage(id, []byte("abc"))
		if err := pub.Channel("delay-channel").Topic("order-topic").PublishInSequence(msg, "aaa"); err != nil {
			return msg, err
		}
		msg, err := pub.Wait(ctx, id)
		if err != nil {
			return msg, err
		}
		return msg, nil
	}).AddCargo(xid.New().String(), func(ctx context.Context, id string) (*beanq.Message, error) {
		msg := beanq.NewMessage(id, []byte("cde"))
		if err := pub.Channel("delay-channel").Topic("order-topic").PublishInSequence(msg, "aaa"); err != nil {
			return msg, err
		}
		msg, err := pub.Wait(ctx, id)
		if err != nil {
			return msg, err
		}
		return msg, nil
	})

	err := pipe.ExecuteWithContext(ctx)
	if err != nil {
		panic(err)
	}
}
