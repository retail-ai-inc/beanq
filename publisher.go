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
	// get message
	message := beanq.NewMessage(d)
	pub := beanq.NewPublisher()
	err := pub.Publish(message, opt.Topic("ch2"), opt.Channel("g2"),opt.Retry(3),opt.MaxLen(100),opt.Priority(10))
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

	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/logger"
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

func NewPublisher(config BeanqConfig) *pubClient {
	opts := DefaultOptions

	publisherOnce.Do(func() {

		if config.PoolSize != 0 {
			opts.PoolSize = config.PoolSize
		}

		pool, err := ants.NewPool(opts.PoolSize, ants.WithPreAlloc(true))
		if err != nil {
			logger.New().With("", err).Fatal("goroutine pool error")
		}
		Config = config
		if config.Driver == "redis" {
			beanqPublisher = &pubClient{
				broker: NewRedisBroker(pool, config),
				wg:     nil,
			}
		} else {
			// Currently beanq is only supporting `redis` driver other than that return `nil` beanq client.
			beanqPublisher = nil
		}
	})

	return beanqPublisher
}

func (t *pubClient) PublishWithContext(ctx context.Context, msg *Message, option ...OptionI) error {
	opts, err := ComposeOptions(option...)
	if err != nil {
		return err
	}

	msg.Values["topic"] = opts.Topic
	msg.Values["channel"] = opts.Channel
	msg.Values["retry"] = opts.Retry
	msg.Values["priority"] = opts.Priority
	msg.Values["maxLen"] = opts.MaxLen
	msg.Values["executeTime"] = opts.ExecuteTime

	return t.broker.enqueue(ctx, msg, opts)
}

func (t *pubClient) DelayPublish(msg *Message, delayTime time.Time, option ...OptionI) error {
	option = append(option, ExecuteTime(delayTime))
	return t.Publish(msg, option...)
}

func (t *pubClient) Publish(msg *Message, option ...OptionI) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return t.PublishWithContext(ctx, msg, option...)
}

func (t *pubClient) Close() error {
	return t.broker.close()
}
