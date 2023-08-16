package beanq

import (
	"testing"

	"beanq/helper/stringx"
	"github.com/spf13/cast"
)

var (
	queue = "ch"
)

func TestConsumer(t *testing.T) {
	csm := NewConsumer()
	csm.Register("group-one", queue, func(task *Task) error {
		Logger.Info(stringx.ByteToString(task.Payload()))
		return nil
	})
	csm.Register("delay-group", "delay-ch", func(task *Task) error {
		Logger.Info(stringx.ByteToString(task.Payload()))
		return nil
	})
	csm.StartConsumer()
}

func TestConsumerSingle(t *testing.T) {
	csm := NewConsumer()
	csm.Register("g"+cast.ToString(1), "ch2", func(task *Task) error {
		Logger.Info(stringx.ByteToString(task.Payload()))
		return nil
	})
	csm.StartConsumer()
}

func TestConsumerMultiple(t *testing.T) {
	csm := NewConsumer()
	for i := 0; i < 5; i++ {

		csm.Register("g"+cast.ToString(i), "ch2", func(task *Task) error {
			Logger.Info(stringx.ByteToString(task.Payload()))

			return nil
		})
		csm.Register("g"+cast.ToString(i), "ch2", func(task *Task) error {
			Logger.Info(stringx.ByteToString(task.Payload()))

			return nil
		})
	}
	csm.StartConsumer()
}
