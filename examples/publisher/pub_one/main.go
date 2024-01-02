package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/helper/json"
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
	pubOneInfo()
}

func pubOneInfo() {
	// msg can struct or map
	msg := struct {
		Id   int
		Info string
	}{
		1,
		"msg------1",
	}

	d, _ := json.Marshal(msg)
	// get message
	bmsg := beanq.NewMessage(d)
	config := initCnf()
	pub := beanq.NewPublisher(config)
	err := pub.Publish(bmsg, beanq.Topic("ch2"), beanq.Channel("g2"))
	if err != nil {

	}
	defer pub.Close()

	// publish information
	fmt.Printf("SendMsgs：%+v \n", bmsg)
}
