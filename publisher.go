package beanq

import (
	"context"
	"sync"
	"time"

	"beanq/helper/file"
	"beanq/internal/base"
	opt "beanq/internal/options"

	"github.com/labstack/gommon/log"
)

type client struct {
	broker Broker
	wg     *sync.WaitGroup
}

var _ BeanqPub = new(client)

var (
	beanqPublisherOnce sync.Once
	beanqPublisher     *client
)

func NewPublisher() *client {

	beanqPublisherOnce.Do(func() {
		initEnv()
		// Initialize the beanq consumer log
		Logger = log.New(Config.Queue.Redis.Prefix)

		// IMPORTANT: Configure debug log. If `path` is empty then push the log into `stdout`.
		if Config.Queue.DebugLog.Path != "" {
			if file, err := file.OpenFile(Config.Queue.DebugLog.Path); err != nil {
				Logger.Errorf("Unable to open log file: %v", err)
				beanqPublisher = nil
				return
			} else {
				Logger.SetOutput(file)
			}
		}

		// Set the default log level as DEBUG.
		Logger.SetLevel(log.DEBUG)

		if Config.Queue.Driver == "redis" {
			beanqPublisher = &client{
				broker: NewRedisBroker(Config),
				wg:     nil,
			}
		} else {
			// Currently beanq is only supporting `redis` driver other than that return `nil` beanq client.
			beanqPublisher = nil
		}
	})

	return beanqPublisher
}

func (t *client) PublishWithContext(ctx context.Context, task *Task, option ...opt.OptionI) error {

	opts, err := opt.ComposeOptions(option...)
	if err != nil {
		return err
	}

	task.Values["queue"] = opts.Queue
	task.Values["group"] = opts.Group
	task.Values["retry"] = opts.Retry
	task.Values["priority"] = opts.Priority
	task.Values["maxLen"] = opts.MaxLen
	task.Values["executeTime"] = opts.ExecuteTime

	return t.broker.enqueue(ctx, base.MakeZSetKey(opts.Group, opts.Queue), task, opts)

}

func (t *client) DelayPublish(task *Task, delayTime time.Time, option ...opt.OptionI) error {
	option = append(option, opt.ExecuteTime(delayTime))
	return t.Publish(task, option...)
}

func (t *client) Publish(task *Task, option ...opt.OptionI) error {

	return t.PublishWithContext(context.Background(), task, option...)

}

func (t *client) Close() error {
	return t.broker.close()
}
