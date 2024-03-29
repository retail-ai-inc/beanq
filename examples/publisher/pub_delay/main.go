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
	ntime := time.Now()
	for i := 0; i < 1; i++ {

		if time.Now().Sub(ntime).Minutes() >= 1 {
			break
		}

		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)

		b, _ := json.Marshal(m)

		msg := beanq.NewMessage(b)
		delayT := ntime.Add(10 * time.Second)
		if i == 2 {
			delayT = ntime
		}

		if i == 4 {
			y = 8
		}
		if i == 3 {
			y = 10
			delayT = ntime.Add(35 * time.Second)

		}
		// fmt.Println(delayT)
		// continue
		if err := pub.DelayPublish(msg, delayT, beanq.Topic("delay-topic"), beanq.Channel("delay-channel"), beanq.Priority(float64(y))); err != nil {
			log.Fatalln(err)
		}
		// if err := pub.Publish(msg, beanq.Topic("delay-ch2"), beanq.Channel("delay-channel")); err != nil {
		// 	log.Fatalln(err)
		// }
		// pub.Publish(task, beanq.Topic("ch2"), beanq.Channel("g2"))
	}
	// pub.Publish(beanq.NewMessage([]byte("aaa")), beanq.Channel("channel1"), beanq.Topic("topic1"))
	defer pub.Close()
}
