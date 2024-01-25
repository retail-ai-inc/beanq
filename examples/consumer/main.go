package main

import (
	"log"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"sync"

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
func main() {

	config := initCnf()
	// register consumer
	csm := beanq.NewConsumer(config)
	// register normal consumer
	csm.Register("g2", "ch2", func(msg *beanq.Message) error {
		// TODO:logic
		// like this:
		// time.Sleep(3 * time.Second) // this is my business.
		logger.New().With("g2", "ch2").Info(msg.Payload())

		return nil
	})
	// register delay consumer
	csm.Register("delay-channel", "delay-topic", func(msg *beanq.Message) error {
		logger.New().With("delay-channel", "delay-topic").Info(msg.Payload())
		panic("this is a panic")
		return nil
	})
	csm.Register("delay-channel", "delay-ch2", func(msg *beanq.Message) error {
		logger.New().With("delay-channel", "delay-ch2").Info(msg.Payload())
		return nil
	})
	csm.Register("default-channel", "BatchCartStateTimoutJobHandler", func(msg *beanq.Message) error {
		logger.New().With("default-channel", "BatchCartStateTimoutJobHandler").Info(msg.Payload())
		return nil
	})
	csm.Register("default-channel", "default-topic", func(message *beanq.Message) error {
		logger.New().With("default-channel", "default-topic").Info(message.Payload())
		return nil
	})
	// start ping
	csm.StartPing()
	// begin to consume information
	csm.StartConsumer()
}
