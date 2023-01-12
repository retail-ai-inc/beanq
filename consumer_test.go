package beanq

import (
	"testing"

	"github.com/spf13/cast"
)

var (
	queue = "ch"
)

/*
  - TestConsumer
  - @Description:
    consumer
  - @param t
*/
func TestConsumer(t *testing.T) {

	server := NewServer()

	server.Register("group-one", queue, func(task *Task) error {
		Logger.Info(task.Payload())
		return nil
	})
	server.Register("delay-group", "delay-ch", func(task *Task) error {
		Logger.Info(task.Payload())
		return nil
	})

	csm := NewConsumer()
	csm.StartConsumer(server)

}
func TestConsumerSingle(t *testing.T) {

	server := NewServer()
	server.Register("g"+cast.ToString(1), "ch2", func(task *Task) error {
		Logger.Info(task.Payload())
		return nil
	})
	csm := NewConsumer()

	csm.StartConsumer(server)
}

func TestConsumerMultiple(t *testing.T) {
	server := NewServer()
	for i := 0; i < 5; i++ {

		server.Register("g"+cast.ToString(i), "ch2", func(task *Task) error {
			Logger.Info(task.Payload())

			return nil
		})
		server.Register("g"+cast.ToString(i), "ch2", func(task *Task) error {
			Logger.Info(task.Payload())

			return nil
		})
	}

	csm := NewConsumer()

	csm.StartConsumer(server)
}
