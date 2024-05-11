package main

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/spf13/cast"
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
	pub := beanq.New(config)

	m := make(map[string]any)
	ctx := context.Background()
	for i := 0; i < 20; i++ {
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)

		if err := pub.BQ().WithContext(ctx).PublishInSequential("delay-channel", "order-topic", b); err != nil {
			logger.New().Error(err)
		}

	}

}
