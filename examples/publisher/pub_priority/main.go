package main

import (
	"log"
	"time"

	"beanq"
	"beanq/helper/json"
	opt "beanq/internal/options"
	"github.com/spf13/cast"
)

func main() {
	pubMoreAndPriorityInfo()
}

func pubMoreAndPriorityInfo() {
	pub := beanq.NewPublisher()
	m := make(map[string]string)

	for i := 0; i < 5; i++ {
		var y float64 = 0
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)

		task := beanq.NewTask(b, beanq.SetName("update"))
		delayT := time.Now().Add(10 * time.Second)

		if i == 3 {
			y = 10
		}
		if err := pub.DelayPublish(task, delayT, opt.Queue("delay-ch"), opt.Group("delay-group"), opt.Priority(y)); err != nil {
			log.Fatalln(err)
		}
	}
	defer pub.Close()
}
