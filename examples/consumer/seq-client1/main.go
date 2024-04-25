package main

import (
	"fmt"
	"log"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
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

type seqCustomer struct {
	a string
}

func (t *seqCustomer) Handle(ctx context.Context, message *beanq.Message) error {
	logger.New().Info(message)
	return nil
}
func (t *seqCustomer) Cancel(ctx context.Context, message *beanq.Message) error {
	return nil
}
func (t *seqCustomer) Error(ctx context.Context, err error) {

}

func main() {

	// register consumer

	config := initCnf()
	csm := beanq.NewConsumer(config)

	// register sequential consumer
	// csm.SubscribeSequential("", "", &seqCustomer{})
	csm.SubscribeSequential("delay-channel", "order-topic", beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, message *beanq.Message) error {
			fmt.Printf("result:%+v,time:%+v \n", message, time.Now())
			return nil
		},
		DoCancel: func(ctx context.Context, message *beanq.Message) error {
			return nil
		},
		DoError: func(ctx context.Context, err error) {

		},
	})
	// begin to consume information
	csm.StartConsumer()

}
