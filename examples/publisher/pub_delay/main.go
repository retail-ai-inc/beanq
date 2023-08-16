package main

import (
	"log"
	"time"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/helper/json"
	opt "github.com/retail-ai-inc/beanq/internal/options"
	"github.com/spf13/cast"
)

func main() {
	pubDelayInfo()
}

func pubDelayInfo() {
	pub := beanq.NewPublisher()

	m := make(map[string]any)
	now := time.Now()
	for i := 0; i < 10; i++ {
		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)
		m["a"] = "sfdsf"
		m["b"] = "bbbb"
		m["c"] = "ccccc"
		m["d"] = "sdfsfdsfsf"
		m["e"] = "sdfsfsfsf"

		b, _ := json.Marshal(m)

		task := beanq.NewTask(b, beanq.SetName("update"))
		delayT := now.Add(10 * time.Second)
		if i == 2 {
			delayT = now
		}
		if i == 3 {
			y = 10
			delayT = now.Add(35 * time.Second)
		}

		if err := pub.DelayPublish(task, delayT, opt.Queue("delay-ch"), opt.Group("delay-group"), opt.Priority(float64(y))); err != nil {
			log.Fatalln(err)
		}
	}

	defer pub.Close()
}
