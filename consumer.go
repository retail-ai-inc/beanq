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

	"github.com/retail-ai-inc/beanq/helper/logger"
)

type ErrorCallback func(msg *Message, err error) error

type Consumer struct {
	broker         Broker
	opts           *Options
	mu             *sync.RWMutex
	timeOut        time.Duration
	errorCallbacks []ErrorCallback
}

var _ BeanqSub = (*Consumer)(nil)
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

	Config.Store(config)
	beanqConsumer = &Consumer{
		broker:  NewBroker(config),
		mu:      new(sync.RWMutex),
		timeOut: timeOut,
	}

	return beanqConsumer
}

// Subscribe
// Subscribe the channel and topic to be consumed
//
//	@Description:
//
//	@receiver t
//	@param channel
//	@param topic
//	@param consumerFun
func (t *Consumer) Subscribe(channelName, topicName string, subscribe ConsumerFunc) {
	t.subscribe(normalSubscribe, channelName, topicName, subscribe)
}

func (t *Consumer) SubscribeSequential(channelName, topicName string, consumer ConsumerFunc) {
	t.subscribe(sequentialSubscribe, channelName, topicName, consumer)
}

func (t *Consumer) subscribe(subType subscribeType, channelName, topicName string, subscribe ConsumerFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if channelName == "" {
		channelName = DefaultOptions.DefaultChannel
	}
	if topicName == "" {
		topicName = DefaultOptions.DefaultTopic
	}

	t.broker.addConsumer(subType, channelName, topicName, subscribe)
}

func (t *Consumer) WithErrorCallBack(callbacks ...ErrorCallback) *Consumer {
	clone := *t
	clone.errorCallbacks = append(t.errorCallbacks, callbacks...)
	return &clone
}

func (t *Consumer) StartConsumerWithContext(ctx context.Context) {
	t.broker.startConsuming(ctx)
}

func (t *Consumer) StartConsumer() {
	t.ping()
	t.StartConsumerWithContext(context.Background())
}

func (t *Consumer) ping() {
	health := Config.Load().(BeanqConfig).Health

	if health.Host == "" || health.Port == "" {
		return
	}

	go func() {
		hdl := &http.ServeMux{}
		hdl.HandleFunc("/ping", func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte("Beanq 🚀  pong"))
			return
		})

		srv := &http.Server{
			Addr:    strings.Join([]string{health.Host, health.Port}, ":"),
			Handler: hdl,
		}
		logger.New().Info("Start Ping On:", health.Host, ":", health.Port)
		if err := srv.ListenAndServe(); err != nil {
			logger.New().Fatal(err)
		}
	}()
}
