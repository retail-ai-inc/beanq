package main

import (
	"log"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/retail-ai-inc/beanq"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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
	csm.Register("g2", "ch2", func(task *beanq.Task) error {
		// TODO:logic
		// like this:
		// time.Sleep(3 * time.Second) // this is my business.
		beanq.Logger.Info(task.Payload(), zap.String("g2", "ch2"))

		return nil
	})
	// register delay consumer
	csm.Register("delay-group", "delay-ch", func(task *beanq.Task) error {
		beanq.Logger.Info(task.Payload(), zap.String("delay-group", "delay-ch"))

		return nil
	})
	csm.Register("default-group", "BatchCartStateTimoutJobHandler", func(task *beanq.Task) error {
		beanq.Logger.Info(task.Payload())
		return nil
	})
	// start ping
	csm.StartPing()
	// begin to consume information
	csm.StartConsumer()
}
