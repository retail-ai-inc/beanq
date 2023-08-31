package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/helper/json"
	opt "github.com/retail-ai-inc/beanq/internal/options"
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

		b, _ := json.Marshal(m)

		task := beanq.NewTask(b, beanq.SetName("update"))
		delayT := ntime.Add(10 * time.Second)
		if i == 2 {
			delayT = ntime
		}
		fmt.Printf("---:%+v \n", delayT.Format("2006-01-02 15:04:05 "))
		if i == 4 {
			y = 8
		}
		if i == 3 {
			y = 10
			delayT = ntime.Add(35 * time.Second)
			fmt.Printf("=====%+v \n", delayT.Format("2006-01-02 15:04:05"))
		}

		if err := pub.DelayPublish(task, delayT, opt.Queue("delay-ch"), opt.Group("delay-group"), opt.Priority(float64(y))); err != nil {
			log.Fatalln(err)
		}
	}

	defer pub.Close()
}
