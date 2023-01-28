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

// Package beanq
// @Description:
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

type pubClient struct {
	broker Broker
	wg     *sync.WaitGroup
}

var _ BeanqPub = new(pubClient)

var (
	beanqPublisherOnce sync.Once
	beanqPublisher     *pubClient
)

func NewPublisher() *pubClient {

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
			beanqPublisher = &pubClient{
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

// PublishWithContext
//
//	@Description:
//
// publish jobs
//
//	@receiver t
//	@param ctx
//	@param task
//	@param option
//	@return error
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

	return t.broker.enqueue(ctx, base.MakeZSetKey(opts.Group, opts.Queue), task, opts)

}

// DelayPublish
//
//	@Description:
//
// publish delay job
//
//	@receiver t
//	@param task
//	@param delayTime
//	@param option
//	@return error
func (t *pubClient) DelayPublish(task *Task, delayTime time.Time, option ...opt.OptionI) error {
	option = append(option, opt.ExecuteTime(delayTime))
	return t.Publish(task, option...)
}

// Publish
//
//	@Description:
//
// publish job
//
//	@receiver t
//	@param task
//	@param option
//	@return error
func (t *pubClient) Publish(task *Task, option ...opt.OptionI) error {

	return t.PublishWithContext(context.Background(), task, option...)

}

// Close
//
//	@Description:
//	@receiver t
//	@return error
func (t *pubClient) Close() error {
	return t.broker.close()
}
