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
	msg := struct {
		Id   int
		Info string
	}{
		1,
		"msg",
	}

	d, _ := json.Marshal(msg)
	// get task
	task := beanq.NewTask(d)
	pub := beanq.NewPublisher()
	err := pub.Publish(task, opt.Queue("ch2"), opt.Group("g2"),opt.Retry(3),opt.MaxLen(100),opt.Priority(10))
	if err != nil {
		Logger.Error(err)
	}
	defer pub.Close()
*/

package beanq

import (
	"context"
	"sync"
	"time"

	"beanq/helper/logger"
	opt "beanq/internal/options"
	"go.uber.org/zap"

	"github.com/panjf2000/ants/v2"
)

type pubClient struct {
	broker Broker
	wg     *sync.WaitGroup
}

var _ BeanqPub = new(pubClient)

var (
	publisherOnce  sync.Once
	beanqPublisher *pubClient
)

func NewPublisher() *pubClient {
	opts := opt.DefaultOptions

	publisherOnce.Do(func() {
		initEnv()

		param := make([]logger.LoggerInfoFun, 0)
		// IMPORTANT: Configure debug log. If `path` is empty then push the log into `stdout`.
		if Config.Queue.DebugLog.Path != "" {
			param = append(param, logger.WithInfoFile(Config.Queue.DebugLog.Path))
		}
		// Initialize the beanq consumer log
		Logger = logger.InitLogger(param...).With(zap.String("prefix", Config.Queue.Redis.Prefix))

		if Config.Queue.PoolSize != 0 {
			opts.PoolSize = Config.Queue.PoolSize
		}

		pool, err := ants.NewPool(opts.PoolSize, ants.WithPreAlloc(true))
		if err != nil {
			Logger.Fatal("goroutine pool error", zap.Error(err))
		}

		if Config.Queue.Driver == "redis" {
			beanqPublisher = &pubClient{
				broker: NewRedisBroker(pool, Config),
				wg:     nil,
			}
		} else {
			// Currently beanq is only supporting `redis` driver other than that return `nil` beanq client.
			beanqPublisher = nil
		}
	})

	return beanqPublisher
}

func (t *pubClient) PublishWithContext(ctx context.Context, task *Task, option ...opt.OptionI) error {
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

	return t.broker.enqueue(ctx, task, opts)
}

func (t *pubClient) DelayPublish(task *Task, delayTime time.Time, option ...opt.OptionI) error {
	option = append(option, opt.ExecuteTime(delayTime))
	return t.Publish(task, option...)
}

func (t *pubClient) Publish(task *Task, option ...opt.OptionI) error {
	return t.PublishWithContext(context.Background(), task, option...)
}

func (t *pubClient) Close() error {
	return t.broker.close()
}
