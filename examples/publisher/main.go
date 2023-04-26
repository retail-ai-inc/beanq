package main

import (
	"fmt"
	"log"
	"time"

	"beanq"
	"beanq/helper/json"
	opt "beanq/internal/options"
	"github.com/spf13/cast"
)

func main() {
	// pubOneInfo()
	// pubMoreInfo()
	pubDelayInfo()
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
	// get task
	task := beanq.NewTask(d)
	pub := beanq.NewPublisher()
	err := pub.Publish(task, opt.Queue("ch2"), opt.Group("g2"))
	if err != nil {

	}
	defer pub.Close()

	// publish information
	fmt.Printf("SendMsgs：%+v \n", task)
}

func pubMoreInfo() {
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

func pubDelayInfo() {
	pub := beanq.NewPublisher()

	m := make(map[string]string)

	for i := 0; i < 10; i++ {
		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)

		task := beanq.NewTask(b, beanq.SetName("update"))
		delayT := time.Now().Add(10 * time.Second)
		if i == 2 {
			delayT = time.Now()
		}
		if i == 3 {
			y = 10
			delayT = time.Now().Add(35 * time.Minute)
		}

		// This part is for the convenience of future testing, so keep it for now
		/*
			if i > 20 {
				delayT = time.Now().Add(25 * time.Second)
			}*/
		if err := pub.DelayPublish(task, delayT, opt.Queue("delay-ch"), opt.Group("delay-group"), opt.Priority(float64(y))); err != nil {
			log.Fatalln(err)
		}
	}

	defer pub.Close()
}
