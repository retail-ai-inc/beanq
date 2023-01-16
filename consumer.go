package beanq

import (
	"context"
	"sync"

	"beanq/helper/file"
	opt "beanq/internal/options"
	"github.com/labstack/gommon/log"
)

type ConsumerHandler struct {
	Group, Queue string
	ConsumerFun  DoConsumer
}

type Consumer struct {
	broker Broker
	opts   *opt.Options
	mu     sync.RWMutex
	m      []*ConsumerHandler
}

var _ BeanqSub = new(Consumer)
var (
	beanqConsumerOnce sync.Once
	beanqConsumer     *Consumer
)

func NewConsumer() *Consumer {
	opts := opt.DefaultOptions

	beanqConsumerOnce.Do(func() {
		initEnv()
		// Initialize the beanq consumer log
		Logger = log.New(Config.Queue.Redis.Prefix)

		// IMPORTANT: Configure debug log. If `path` is empty then push the log into `stdout`.
		if Config.Queue.DebugLog.Path != "" {
			if file, err := file.OpenFile(Config.Queue.DebugLog.Path); err != nil {
				Logger.Errorf("Unable to open log file: %v", err)
				beanqConsumer = nil
				return
			} else {
				Logger.SetOutput(file)
			}
		}

		// Set the default log level as DEBUG.
		Logger.SetLevel(log.DEBUG)

		if Config.Queue.KeepJobsInQueue != 0 {
			opts.KeepJobInQueue = Config.Queue.KeepJobsInQueue
		}

		if Config.Queue.KeepFailedJobsInHistory != 0 {
			opts.KeepFailedJobsInHistory = Config.Queue.KeepFailedJobsInHistory
		}

		if Config.Queue.KeepSuccessJobsInHistory != 0 {
			opts.KeepSuccessJobsInHistory = Config.Queue.KeepSuccessJobsInHistory
		}

		if Config.Queue.MinWorkers != 0 {
			opts.MinWorkers = Config.Queue.MinWorkers
		}

		if Config.Queue.JobMaxRetries != 0 {
			opts.JobMaxRetry = Config.Queue.JobMaxRetries
		}

		if Config.Queue.Driver == "redis" {
			beanqConsumer = &Consumer{
				broker: NewRedisBroker(Config),
				opts:   opts,
				mu:     sync.RWMutex{},
			}
		} else {
			// Currently beanq is only supporting `redis` driver other than that return `nil` beanq client.
			beanqConsumer = nil
		}
	})
	return beanqConsumer
}
func (t *Consumer) Register(group, queue string, consumerFun DoConsumer) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if group == "" {
		group = opt.DefaultOptions.DefaultGroup
	}
	if queue == "" {
		queue = opt.DefaultOptions.DefaultQueueName
	}

	t.m = append(t.m, &ConsumerHandler{
		Group:       group,
		Queue:       queue,
		ConsumerFun: consumerFun,
	})
}
func (t *Consumer) StartConsumerWithContext(ctx context.Context) {

	ctx = context.WithValue(ctx, "options", t.opts)
	t.broker.start(ctx, t.m)

}

func (t *Consumer) StartConsumer() {

	ctx := context.Background()
	t.StartConsumerWithContext(ctx)

}
func (t *Consumer) StartUI() error {
	return nil
}
