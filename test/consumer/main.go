package main

import (
	"fmt"

	"beanq"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
)

func main() {

	redisOpts := &redis.Options{
		Addr:      beanq.Env.Queue.Redis.Host + ":" + cast.ToString(beanq.Env.Queue.Redis.Port),
		Dialer:    nil,
		OnConnect: nil,
		Username:  "",
		Password:  beanq.Env.Queue.Redis.Password,
		DB:        beanq.Env.Queue.Redis.Db,
	}

	server := beanq.NewServer(3)
	server.Register("g2", "ch2", func(task *beanq.Task) error {
		fmt.Printf("PayLoad：%+v \n", task.Payload())
		return nil
	})
	server.Register("delay-group", "delay-ch", func(task *beanq.Task) error {
		fmt.Printf("Delay:%+v \n", task.Payload())
		return nil
	})

	csm := beanq.NewConsumer(beanq.NewRedisBroker(redisOpts), nil)
	csm.StartConsumer(server)
}
