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

var _ BeanqPub = new(Client)

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

	task.Values["queue"] = opts.Queue
	task.Values["group"] = opts.Group
	task.Values["retry"] = opts.Retry
	task.Values["priority"] = opts.Priority
	task.Values["maxLen"] = opts.MaxLen
	task.Values["executeTime"] = opts.ExecuteTime
	return t.broker.enqueue(t.ctx, base.MakeZSetKey(opts.Group, opts.Queue), task, opts)

}

func (t *Client) Close() error {
	return t.broker.close()
}
