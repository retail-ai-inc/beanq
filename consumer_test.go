package beanq

import (
	"fmt"
	"testing"

	"github.com/spf13/cast"
)

/*
  - TestConsumer
  - @Description:
    consumer
  - @param t
*/
func TestConsumer(t *testing.T) {

	server := NewServer(3)
	server.Register("aa", queue, func(task *Task) error {
		fmt.Printf("PayLoad:%+v,Id:%s \n", task.Payload(), task.Id())
		return nil
	})
	server.Register("", "delay-ch", func(task *Task) error {
		fmt.Printf("Delay:%+v,Id:%s \n", task.Payload(), task.Id())
		return nil
	})

	csm := NewConsumer(NewRedisBroker(optionParameter.RedisOptions), nil)
	csm.StartConsumer(server)
}

func TestConsumerMultiple(t *testing.T) {
	server := NewServer(3)
	for i := 0; i < 5; i++ {

		server.Register("g"+cast.ToString(i), "ch2", func(task *Task) error {
			fmt.Printf(cast.ToString(i)+"PayLoad:%+v \n", task.Payload())
			return nil
		})
		server.Register("g"+cast.ToString(i), "ch2", func(task *Task) error {
			Logger.Info(cast.ToString(1)+"PayLoad: %+v", task.Payload())

			return nil
		})
	}

	csm := NewConsumer(NewRedisBroker(optionParameter.RedisOptions), nil)
	csm.StartConsumer(server)
}
