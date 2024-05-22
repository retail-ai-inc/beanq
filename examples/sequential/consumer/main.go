package main

import (
	"context"
	"log"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/helper/logger"
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

type seqCustomer struct {
	metadata string
}

func (t *seqCustomer) Handle(ctx context.Context, message *beanq.Message) error {
	time.Sleep(time.Second * 1)
	log.Printf("%s:%v\n", t.metadata, message)
	return nil
}
func (t *seqCustomer) Cancel(ctx context.Context, message *beanq.Message) error {
	return nil
}
func (t *seqCustomer) Error(ctx context.Context, err error) {

}

func main() {
	config := initCnf()
	csm := beanq.New(config)

	ctx := context.Background()
	_, err := csm.BQ().WithContext(ctx).SubscribeSequential("delay-channel", "order-topic", beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, message *beanq.Message) error {
			time.Sleep(time.Second * 2)
			log.Printf("result:%+v,time:%+v \n", message, time.Now())
			return nil
		},
		DoCancel: func(ctx context.Context, message *beanq.Message) error {
			return nil
		},
		DoError: func(ctx context.Context, err error) {

		},
	})
	if err != nil {
		logger.New().Error(err)
	}

	_, err = csm.BQ().WithContext(ctx).SubscribeSequential("delay-channel", "order-topic", &seqCustomer{
		metadata: "I am a custom",
	})
	if err != nil {
		logger.New().Error(err)
	}
	// begin to consume information
	csm.Wait(ctx)

}
