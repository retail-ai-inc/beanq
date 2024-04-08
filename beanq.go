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
	"sync/atomic"
	"time"
)

type (
	DebugLog struct {
		Path string `json:"path"`
		On   bool   `json:"on"`
	}
	Health struct {
		Port string `json:"port"`
		Host string `json:"host"`
	}
	Redis struct {
		Host               string        `json:"host"`
		Port               string        `json:"port"`
		Password           string        `json:"password"`
		Prefix             string        `json:"prefix"`
		Database           int           `json:"database"`
		MaxLen             int64         `json:"maxLen"`
		MinIdleConnections int           `json:"minIdleConnections"`
		DialTimeout        time.Duration `json:"dialTimeout"`
		ReadTimeout        time.Duration `json:"readTimeout"`
		WriteTimeout       time.Duration `json:"writeTimeout"`
		PoolTimeout        time.Duration `json:"poolTimeout"`
		MaxRetries         int           `json:"maxRetries"`
		PoolSize           int           `json:"poolSize"`
	}
	BeanqConfig struct {
		Health                   Health        `json:"health"`
		DebugLog                 DebugLog      `json:"debugLog"`
		Driver                   string        `json:"driver"`
		Redis                    Redis         `json:"redis"`
		ConsumerPoolSize         int           `json:"consumerPoolSize"`
		JobMaxRetries            int           `json:"jobMaxRetries"`
		DeadLetterIdle           time.Duration `json:"deadLetterIdle"`
		KeepFailedJobsInHistory  time.Duration `json:"keepFailedJobsInHistory"`
		KeepSuccessJobsInHistory time.Duration `json:"keepSuccessJobsInHistory"`
		PublishTimeOut           time.Duration `json:"publishTimeOut"`
		ConsumeTimeOut           time.Duration `json:"consumeTimeOut"`
		MinConsumers             int64         `json:"minConsumers"`
	}
)

// Config Hold the useful configuration settings of beanq so that we can use it quickly from anywhere.
var Config atomic.Value

type BeanqPub interface {
	Publish(msg *Message, option ...OptionI) error
	PublishWithContext(ctx context.Context, msg *Message, option ...OptionI) error
	PublishWithDelay(msg *Message, delayTime time.Time, option ...OptionI) error
	PublishInSequence(msg *Message, orderKey string, option ...OptionI) error
}

type BeanqSub interface {
	Subscribe(channel, topic string, subscribe RunSubscribe)
	StartConsumer()
	StartConsumerWithContext(ctx context.Context)
	ping()
}
type IHandle interface {
	Check(ctx context.Context) error
	Work(ctx context.Context, done <-chan struct{})
	DeadLetter(ctx context.Context, claimDone <-chan struct{}) error
}
