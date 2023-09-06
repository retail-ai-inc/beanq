package main

import (
	"fmt"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/helper/json"
	opt "github.com/retail-ai-inc/beanq/internal/options"
)

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
