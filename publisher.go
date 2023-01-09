package beanq

import (
	"context"
	"sync"
	"time"

	"beanq/helper/file"
	"beanq/internal/base"
	opt "beanq/internal/options"

	"github.com/labstack/gommon/log"
	"github.com/spf13/viper"
)

type Client struct {
	broker Broker
	ctx    context.Context
	wg     *sync.WaitGroup
}

var _ BeanqPub = new(Client)

var (
	beanqClientOnce sync.Once
	beanqClient     *Client
)

func NewClient() *Client {

	beanqClientOnce.Do(func() {
		viper.AddConfigPath(".")
		viper.SetConfigType("json")
		viper.SetConfigName("env")

		// Initialize the beanq consumer log
		Logger = log.New("beanq")

		if err := viper.ReadInConfig(); err != nil {
			Logger.Errorf("Unable to open beanq env.json file: %v", err)
			beanqClient = nil
			return
		}

		// IMPORTANT: Unmarshal the env.json into global Config object.
		if err := viper.Unmarshal(&Config); err != nil {
			Logger.Errorf("Unable to unmarshal the beanq env.json file: %v", err)
			beanqClient = nil
			return
		}

		// IMPORTANT: Configure debug log. If `path` is empty then push the log into `stdout`.
		if Config.Queue.DebugLog.Path != "" {
			if file, err := file.OpenFile(Config.Queue.DebugLog.Path); err != nil {
				Logger.Errorf("Unable to open log file: %v", err)
				beanqClient = nil
				return
			} else {
				Logger.SetOutput(file)
			}
		}

		// Set the default log level as DEBUG.
		Logger.SetLevel(log.DEBUG)

		if Config.Queue.Driver == "redis" {
			beanqClient = &Client{
				broker: NewRedisBroker(Config),
				ctx:    context.Background(),
				wg:     nil,
			}
		} else {
			// Currently beanq is only supporting `redis` driver other than that return `nil` beanq client.
			beanqClient = nil
		}
	})

	return beanqClient
}

func (t *Client) PublishWithContext(ctx context.Context, task *Task, option ...opt.OptionI) (*opt.Result, error) {
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
