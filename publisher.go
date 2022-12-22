package beanq

import (
	"context"
	"sync"
	"time"

	"beanq/internal/base"
	opt "beanq/internal/options"
)

type Client struct {
	broker Broker
	ctx    context.Context
	wg     *sync.WaitGroup
}

func NewClient(broker Broker) *Client {
	return &Client{
		broker: broker,
		ctx:    context.Background(),
		wg:     nil,
	}
}
func (t *Client) PublishContext(ctx context.Context, task *Task, option ...opt.OptionI) (*opt.Result, error) {
	t.ctx = ctx
	return t.Publish(task, option...)
}
func (t *Client) DelayPublish(task *Task, delayTime time.Time, option ...opt.OptionI) (*opt.Result, error) {
	option = append(option, opt.ExecuteTime(delayTime))
	return t.Publish(task, option...)
}
func (t *Client) Publish(task *Task, option ...opt.OptionI) (*opt.Result, error) {

	opts, err := opt.ComposeOptions(option...)
	if err != nil {
		return nil, err
	}
	values := base.ParseArgs(opts.Queue, task.Name(), task.Payload(), opts.Retry, opts.MaxLen, opts.ExecuteTime)
	return t.broker.Enqueue(t.ctx, values, opts)

}
func (t *Client) Close() error {
	return t.broker.Close()
}
