package main

import (
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
}

func (t *seqCustomer) Run(message *beanq.Message) error {
	time.Sleep(5 * time.Second)
	logger.New().Info(message)
	return nil
}
func (t *seqCustomer) Cancel(message *beanq.Message) error {
	return nil
}
func (t *seqCustomer) Error(err error) {

}

func main() {

	// register consumer

	config := initCnf()
	csm := beanq.NewConsumer(config)
	// register normal consumer
	// csm.Register("g2", "ch2", func(ctx context.Context, msg *beanq.Message) error {
	// 	// TODO:logic
	// 	// like this:
	// 	// time.Sleep(3 * time.Second) // this is my business.
	// 	logger.New().With("g2", "ch2").Info(msg.Payload())
	//
	// 	return nil
	// })
	// register delay consumer

	csm.SubscribeSequential("delay-channel", "order-topic", &seqCustomer{})
	// begin to consume information
	csm.StartConsumer()

}
