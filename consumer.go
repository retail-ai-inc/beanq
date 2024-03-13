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

package beanq

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/logger"
)

type ConsumerHandler struct {
	IHandle
	ConsumerFun DoConsumer
	Channel     string
	Topic       string
}

type Consumer struct {
	broker  Broker
	opts    *Options
	m       []*ConsumerHandler
	mu      sync.RWMutex
	timeOut time.Duration
}

var _ BeanqSub = new(Consumer)
var (
	beanqConsumer *Consumer
)

func NewConsumer(config BeanqConfig) *Consumer {

	poolSize := DefaultOptions.ConsumerPoolSize
	if config.ConsumerPoolSize != 0 {
		poolSize = config.ConsumerPoolSize
	}
	config.ConsumerPoolSize = poolSize

	timeOut := DefaultOptions.ConsumeTimeOut
	if config.ConsumeTimeOut > 0 {
		timeOut = config.ConsumeTimeOut
	}
	config.ConsumeTimeOut = timeOut

	pool, err := ants.NewPool(poolSize, ants.WithPreAlloc(true))
	if err != nil {
		logger.New().With("", err).Fatal("goroutine pool error")
	}
	Config.Store(config)
	if config.Driver == "redis" {
		beanqConsumer = &Consumer{
			broker:  newRedisBroker(pool),
			mu:      sync.RWMutex{},
			timeOut: timeOut,
		}
	} else {
		// Currently beanq is only supporting `redis` driver other than that return `nil` beanq client.
		beanqConsumer = nil
	}

	return beanqConsumer
}

// Register
// Register the channel and topic to be consumed
//
//	@Description:
//
//	@receiver t
//	@param channel
//	@param topic
//	@param consumerFun
func (t *Consumer) Register(channelName, topicName string, consumerFun DoConsumer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if channelName == "" {
		channelName = DefaultOptions.DefaultChannel
	}
	if topicName == "" {
		topicName = DefaultOptions.DefaultTopic
	}

	t.m = append(t.m, &ConsumerHandler{
		Channel:     channelName,
		Topic:       topicName,
		ConsumerFun: consumerFun,
	})
}
func (t *Consumer) StartConsumerWithContext(ctx context.Context) {

	t.broker.start(ctx, t.m)

}

func (t *Consumer) StartConsumer() {

	t.StartConsumerWithContext(context.Background())

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
			Addr:    strings.Join([]string{Config.Load().(BeanqConfig).Health.Host, Config.Load().(BeanqConfig).Health.Port}, ":"),
			Handler: hdl,
		}
		if err := srv.ListenAndServe(); err != nil {
			logger.New().With("", err).Error("ping server error")
		}
	}()

	return nil
}
