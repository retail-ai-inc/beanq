package main

import (
	"beanq"
)

func main() {
	// registe consumer
	csm := beanq.NewConsumer()
	// registe normal consumer
	csm.Register("g2", "ch2", func(task *beanq.Task) error {
		//todo:logic
		beanq.Logger.Info(task.Payload())
		return nil
	})
	// registe delay consumer
	csm.Register("delay-group", "delay-ch", func(task *beanq.Task) error {
		beanq.Logger.Info(task.Payload())
		return nil
	})
	// begin to consume informations
	csm.StartConsumer()
}
