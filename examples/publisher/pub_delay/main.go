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
	for i := 0; i < 100; i++ {

		if time.Now().Sub(ntime).Minutes() >= 1 {
			break
		}

		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)

		b, _ := json.Marshal(m)

		task := beanq.NewTask(b, beanq.SetName("update"))
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

		if err := pub.DelayPublish(task, delayT, beanq.Queue("delay-ch"), beanq.Group("delay-group"), beanq.Priority(float64(y))); err != nil {
			log.Fatalln(err)
		}
		// pub.Publish(task, beanq.Queue("ch2"), beanq.Group("g2"))
	}
	pub.Publish(beanq.NewTask([]byte("aaa")), beanq.Group("group1"), beanq.Queue("queue1"))
	defer pub.Close()
}
