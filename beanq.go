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
	Register(group, queue string, consumerFun DoConsumer)
	StartConsumer()
	StartConsumerWithContext(ctx context.Context)
	StartUI() error
}
