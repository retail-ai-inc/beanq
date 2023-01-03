package main

import (
	"beanq"

	"github.com/go-redis/redis/v8"
)

func main() {

	server := beanq.NewServer()
	server.Register("g2", "ch2", func(task *beanq.Task, r *redis.Client) error {
		beanq.Logger.Info("PayLoad：%+v", task.Payload())
		return nil
	})
	server.Register("delay-group", "delay-ch", func(task *beanq.Task, r *redis.Client) error {
		beanq.Logger.Info("Delay:%+v", task.Payload())
		return nil
	})

	csm := beanq.NewConsumer(beanq.NewRedisBroker(beanq.Config), nil)
	csm.StartConsumer(server)
}
