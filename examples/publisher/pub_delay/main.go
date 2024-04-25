package main

import (
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/spf13/cast"
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
	runtime.GOMAXPROCS(2)
	pubDelayInfo()
}

func pubDelayInfo() {
	config := initCnf()
	pub := beanq.NewPublisher(config)

	m := make(map[string]any)

	now := time.Now()
	for i := 0; i < 10; i++ {

		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)

		b, _ := json.Marshal(m)

		// This method will use msgId as the idempotent basis
		// example:
		// beanq.NewMessage("1",b)
		msg := beanq.NewMessage("", b)

		delayT := now.Add(10 * time.Second)
		// Execute immediately
		if i == 2 {
			delayT = now
		}
		// priority
		if i == 4 {
			y = 8
		}
		// Delay execution by 35 seconds
		if i == 3 {
			delayT = now.Add(35 * time.Second)
		}

		if err := pub.Channel("delay-channel").Topic("delay-topic").PublishAtTime(msg, delayT, beanq.WithPriority(float64(y))); err != nil {
			log.Println(err)
		}

	}
	defer pub.Close()
}
