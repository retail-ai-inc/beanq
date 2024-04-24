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
	pubMoreAndPriorityInfo()
}

func pubMoreAndPriorityInfo() {

	pub := beanq.NewPublisher(initCnf())
	m := make(map[string]string)

	for i := 0; i < 5; i++ {
		var y float64 = 0
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)

		msg := beanq.NewMessage("", b)
		delayT := time.Now().Add(10 * time.Second)

		if i == 3 {
			y = 10
		}
		if err := pub.PublishAtTime(msg, delayT, beanq.WithTopic("delay-topic"), beanq.WithChannel("delay-channel"), beanq.WithPriority(y)); err != nil {
			log.Fatalln(err)
		}
	}
	defer pub.Close()
}
