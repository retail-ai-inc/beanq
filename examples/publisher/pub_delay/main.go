package main

import (
	"log"
	"runtime"
	"time"

	"beanq"
	"beanq/helper/json"
	opt "beanq/internal/options"
	"github.com/spf13/cast"
)

func main() {
	runtime.GOMAXPROCS(2)
	pubDelayInfo()
}

func pubDelayInfo() {
	pub := beanq.NewPublisher()

	m := make(map[string]any)
	ntime := time.Now()
	for i := 0; i < 10; i++ {

		if time.Now().Sub(ntime).Minutes() >= 1 {
			break
		}

		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)
		m["a"] = "sfdsf"
		m["b"] = "bbbb"
		m["c"] = "ccccc"
		m["d"] = "sdfsfdsfsf"
		m["e"] = "sdfsfsfsf"

		b, _ := json.Marshal(m)

		task := beanq.NewTask(b, beanq.SetName("update"))
		delayT := time.Now().Add(10 * time.Second)
		if i == 2 {
			delayT = time.Now()
		}
		if i == 3 {
			y = 10
			delayT = time.Now().Add(35 * time.Second)
		}

		if err := pub.DelayPublish(task, delayT, opt.Queue("delay-ch"), opt.Group("delay-group"), opt.Priority(float64(y))); err != nil {
			log.Fatalln(err)
		}
	}

	defer pub.Close()
}
