package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"
	"sync"

	beanq "github.com/retail-ai-inc/beanq/v4"
	"github.com/spf13/viper"
)

var (
	configOnce sync.Once
	bqConfig   *beanq.BeanqConfig
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
	return bqConfig
}
func main() {
	config := initCnf()
	csm := beanq.New(config)
	csm.ServeHttp(context.Background())
}
