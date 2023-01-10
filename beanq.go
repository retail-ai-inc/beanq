package beanq

import (
	"context"
	"time"

	opt "beanq/internal/options"
)

type BeanqPub interface {
	Publish(task *Task, option ...opt.OptionI) error
	PublishWithContext(ctx context.Context, task *Task, option ...opt.OptionI) error
	DelayPublish(task *Task, delayTime time.Time, option ...opt.OptionI) error
}

type BeanqSub interface {
	StartConsumer(server *Server)
	StartConsumerWithContext(ctx context.Context, srv *Server)
	StartUI() error
}

type Broker interface {
	enqueue(ctx context.Context, stream string, task *Task, options opt.Option) error
	close() error
	start(ctx context.Context, server *Server)
}

// easy publish
// only input Task and set options
func Publish(task *Task, opts ...opt.OptionI) error {

	pub := NewPublisher()
	defer pub.Close()
	err := pub.Publish(task, opts...)

	if err != nil {
		return err
	}

	return nil
}

// easy consume
// heavily  to implement

func Consume(server *Server, opts *opt.Options) error {
	return nil
}
