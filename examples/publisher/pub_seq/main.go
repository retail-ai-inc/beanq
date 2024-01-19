package main

import (
	"encoding/json"
	"log"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/retail-ai-inc/beanq"
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

	config := initCnf()
	pub := beanq.NewPublisher(config)

	m := make(map[string]any)

	for i := 0; i < 5; i++ {
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)
		msg := beanq.NewMessage(b)
		if err := pub.SequentPublish(msg, "aaa"+cast.ToString(i)); err != nil {
			log.Fatalln(err)
		}

		pub.SequentPublish(msg, "aaa---"+cast.ToString(i), beanq.Channel("delay-channel"), beanq.Topic("delay-ch2"))
	}

}
