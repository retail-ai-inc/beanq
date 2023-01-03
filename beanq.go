package beanq

import (
	"time"

	"beanq/helper/logger"
	opt "beanq/internal/options"
)

// This is a global variable to hold the debug logger so that we can log data from anywhere.
var Logger logger.Logger

// Hold the useful configuration settings of beanq so that we can use it quickly from anywhere.
var Config BeanqConfig

type Beanq interface {
	Publish(task *Task, option ...opt.Option) (*opt.Result, error)
	DelayPublish(task *Task, delayTime time.Time, option ...opt.Option) (*opt.Result, error)
	Start(server *Server)
	StartUI() error
	Close() error
}

func Publish(task *Task, opts ...opt.OptionI) error {
	pub := NewClient()
	_, err := pub.Publish(task, opts...)
	if err != nil {
		return err
	}

	defer pub.Close()
	return nil
}

// TODO:
func Consume(server *Server, opts *opt.Options) error {
	return nil
}
