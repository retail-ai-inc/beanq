package main

import (
	"beanq"
)

func main() {

	server := beanq.NewServer()
	server.Register("g2", "ch2", func(task *beanq.Task) error {
		beanq.Logger.Info(task.Payload())
		return nil
	})
	server.Register("delay-group", "delay-ch", func(task *beanq.Task) error {
		beanq.Logger.Info(task.Payload())
		return nil
	})

	csm := beanq.NewConsumer()

	csm.StartConsumer(server)
}
