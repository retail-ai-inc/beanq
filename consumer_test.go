package beanq

import (
	"testing"

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

	server := NewServer()
	server.Register(group, queue, func(task *Task, r *redis.Client) error {
		Logger.Info("PayLoad: %+v", task.Payload())
		return nil
	})
	server.Register("delay-group", "delay-ch", func(task *Task, r *redis.Client) error {
		Logger.Info("Delay: %+v", task.Payload())
		return nil
	})

	csm := NewConsumer(NewRedisBroker(Config), nil)
	csm.StartConsumer(server)

}
func TestConsumerSingle(t *testing.T) {

	server := NewServer()
	server.Register("g1", "ch2", func(task *Task, r *redis.Client) error {
		Logger.Info("Payload 1: %+v", task.Payload())
		return nil
	})
	server.Register("g2", "ch2", func(task *Task, r *redis.Client) error {
		Logger.Info("Payload 2: %+v", task.Payload())
		return nil
	})

	csm := NewConsumer(NewRedisBroker(Config), nil)
	csm.StartConsumer(server)
}
func TestConsumerSingle2(t *testing.T) {

	server := NewServer()
	server.Register("g"+cast.ToString(1), "ch2", func(task *Task, r *redis.Client) error {
		Logger.Info(cast.ToString(1)+"PayLoad: %+v", task.Payload())
		return nil
	})
	csm := NewConsumer(NewRedisBroker(Config), nil)
	csm.StartConsumer(server)
}
func TestConsumerMultiple(t *testing.T) {

	server := NewServer()
	for i := 0; i < 5; i++ {
		server.Register("g"+cast.ToString(i), "ch2", func(task *Task, r *redis.Client) error {
			Logger.Info(cast.ToString(1)+"PayLoad: %+v", task.Payload())
			return nil
		})
	}

	csm := NewConsumer(NewRedisBroker(Config), nil)
	csm.StartConsumer(server)
}
