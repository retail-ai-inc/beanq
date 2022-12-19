package beanq

import (
	"fmt"
	"testing"

	server2 "beanq/server"
	"beanq/task"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
)

/*
  - TestConsumer
  - @Description:
    consumer
  - @param t
*/
func TestConsumer(t *testing.T) {

	rdb := NewBeanq("redis", options)

	server := server2.NewServer(3)
	server.Register(group, queue, func(task *task.Task, r *redis.Client) error {

		fmt.Printf("PayLoad：%+v \n", task.GPayload())
		return nil
	})
	server.Register("delay-group", "delay-ch", func(task *task.Task, r *redis.Client) error {
		fmt.Printf("Delay:%+v \n", task.GPayload())
		return nil
	})
	rdb.Start(server)

}
func TestConsumerSingle(t *testing.T) {
	rdb := NewBeanq("redis", options)
	server := server2.NewServer(3)
	server.Register("g1", "ch2", func(task *task.Task, r *redis.Client) error {
		fmt.Printf("1PayLoad:%+v \n", task.GPayload())
		return nil
	})
	server.Register("g2", "ch2", func(task *task.Task, r *redis.Client) error {
		fmt.Printf("2PayLoad:%+v \n", task.GPayload())
		return nil
	})
	rdb.Start(server)
}
func TestConsumerSingle2(t *testing.T) {
	rdb := NewBeanq("redis", options)
	server := server2.NewServer(3)
	server.Register("g"+cast.ToString(1), "ch2", func(task *task.Task, r *redis.Client) error {
		fmt.Printf(cast.ToString(1)+"PayLoad:%+v \n", task.GPayload())
		return nil
	})
	rdb.Start(server)
}
func TestConsumerMultiple(t *testing.T) {
	rdb := NewBeanq("redis", options)

	server := server2.NewServer(3)
	for i := 0; i < 5; i++ {
		server.Register("g"+cast.ToString(i), "ch2", func(task *task.Task, r *redis.Client) error {
			fmt.Printf(cast.ToString(i)+"PayLoad:%+v \n", task.GPayload())
			return nil
		})
	}

	rdb.Start(server)
}
func TestDelayConsumer(t *testing.T) {
	// rdb := NewRedis(options)
	// rdb.delayConsumer()

}
