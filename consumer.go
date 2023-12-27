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
	"net/http"
	"strings"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/logger"
)

type ConsumerHandler struct {
	Group, Queue string
	ConsumerFun  DoConsumer
}

type Consumer struct {
	broker Broker
	opts   *Options
	mu     sync.RWMutex
	m      []*ConsumerHandler
}

var _ BeanqSub = new(Consumer)
var (
	beanqConsumer *Consumer
)

func NewConsumer(config BeanqConfig) *Consumer {
	opts := DefaultOptions

	if config.KeepJobsInQueue != 0 {
		opts.KeepJobInQueue = config.KeepJobsInQueue
	}

	if config.KeepFailedJobsInHistory != 0 {
		opts.KeepFailedJobsInHistory = config.KeepFailedJobsInHistory
	}

	if config.KeepSuccessJobsInHistory != 0 {
		opts.KeepSuccessJobsInHistory = config.KeepSuccessJobsInHistory
	}

	if config.MinWorkers != 0 {
		opts.MinWorkers = config.MinWorkers
	}

	if config.JobMaxRetries != 0 {
		opts.JobMaxRetry = config.JobMaxRetries
	}
	if config.PoolSize != 0 {
		opts.PoolSize = config.PoolSize
	}

	pool, err := ants.NewPool(opts.PoolSize, ants.WithPreAlloc(true))
	if err != nil {
		logger.New().With("", err).Fatal("goroutine pool error")
	}
	Config = config
	if config.Driver == "redis" {
		beanqConsumer = &Consumer{
			broker: NewRedisBroker(pool, config),
			opts:   opts,
			mu:     sync.RWMutex{},
		}
	} else {
		// Currently beanq is only supporting `redis` driver other than that return `nil` beanq client.
		beanqConsumer = nil
	}

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
		group = DefaultOptions.DefaultGroup
	}
	if queue == "" {
		queue = DefaultOptions.DefaultQueueName
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
func (t *Consumer) StartPing() error {
	go func() {
		hdl := &http.ServeMux{}
		hdl.HandleFunc("/ping", func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte("Beanq 🚀  pong"))
			return
		})
		srv := &http.Server{
			Addr:    strings.Join([]string{Config.Health.Host, Config.Health.Port}, ":"),
			Handler: hdl,
		}
		if err := srv.ListenAndServe(); err != nil {
			logger.New().With("", err).Error("ping server error")
		}
	}()

	return nil
}
