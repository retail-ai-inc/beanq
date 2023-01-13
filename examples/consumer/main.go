package main

import (
	"beanq"
)

func main() {
	// registe consumer
	server := beanq.NewServer()
	// registe normal consumer
	server.Register("g2", "ch2", func(task *beanq.Task) error {
		//todo:logic
		beanq.Logger.Info(task.Payload())
		return nil
	})
	// registe delay consumer
	server.Register("delay-group", "delay-ch", func(task *beanq.Task) error {
		beanq.Logger.Info(task.Payload())
		return nil
	})
	// begin to consume informations
	csm := beanq.NewConsumer()

	csm.StartConsumer(server)
}
