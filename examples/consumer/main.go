package main

import (
	"beanq"
)

func main() {
	// register consumer
	csm := beanq.NewConsumer()
	// register normal consumer
	csm.Register("g2", "ch2", func(task *beanq.Task) error {
		// TODO:logic
		beanq.Logger.Info(task.Payload())
		return nil
	})
	// register delay consumer
	csm.Register("delay-group", "delay-ch", func(task *beanq.Task) error {
		beanq.Logger.Info(task.Payload())
		return nil
	})
	// begin to consume information
	csm.StartConsumer()
}
