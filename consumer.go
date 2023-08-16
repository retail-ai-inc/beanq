// MIT License

// Copyright The RAI Inc.
// The RAI Authors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// EXAMPLE:
/*
	csm := beanq.NewConsumer()
	csm.Register("group_name", "queue_name", func(task *beanq.Task) error {
		// TODO:logic
		beanq.Logger.Info(task.Payload())
		return nil
	})
	csm.StartConsumer()
*/

package beanq

import (
	"context"
	"sync"

	"github.com/retail-ai-inc/beanq/helper/logger"
	opt "github.com/retail-ai-inc/beanq/internal/options"
	"go.uber.org/zap"

	"github.com/panjf2000/ants/v2"
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

		param := make([]logger.LoggerInfoFun, 0)
		// IMPORTANT: Configure debug log. If `path` is empty then push the log into `stdout`.
		if Config.Queue.DebugLog.Path != "" {
			param = append(param, logger.WithInfoFile(Config.Queue.DebugLog.Path))
		}
		// Initialize the beanq consumer log
		Logger = logger.InitLogger(param...).With(zap.String("prefix", Config.Queue.Redis.Prefix))

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
		if Config.Queue.PoolSize != 0 {
			opts.PoolSize = Config.Queue.PoolSize
		}

		pool, err := ants.NewPool(opts.PoolSize, ants.WithPreAlloc(true))
		if err != nil {
			Logger.Fatal("goroutine pool error", zap.Error(err))
		}

		if Config.Queue.Driver == "redis" {
			beanqConsumer = &Consumer{
				broker: NewRedisBroker(pool, Config),
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

// Register
// Register the group and queue to be consumed
//
//	@Description:
//
//	@receiver t
//	@param group
//	@param queue
//	@param consumerFun
func (t *Consumer) Register(group, queue string, consumerFun DoConsumer) {
	t.mu.Lock()
	defer t.mu.Unlock()
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
