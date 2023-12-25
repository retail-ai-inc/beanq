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
	"time"

	"go.uber.org/zap"
)

type (
	DebugLog struct {
		On   bool
		Path string
	}
	Health struct {
		Port string
		Host string
	}
	Redis struct {
		Host               string
		Port               string
		Password           string
		Database           int
		Prefix             string
		MinIdleConnections int
		DialTimeout        time.Duration
		ReadTimeout        time.Duration
		WriteTimeout       time.Duration
		PoolTimeout        time.Duration
	}
	BeanqConfig struct {
		Driver                   string
		PoolSize                 int
		JobMaxRetries            int
		KeepJobsInQueue          time.Duration
		KeepFailedJobsInHistory  time.Duration
		KeepSuccessJobsInHistory time.Duration
		MinWorkers               int64
		DebugLog
		Redis
		Health
	}
)

// This is a global variable to hold the debug logger so that we can log data from anywhere.
var Logger *zap.Logger

// Hold the useful configuration settings of beanq so that we can use it quickly from anywhere.
var Config BeanqConfig

type BeanqPub interface {
	Publish(task *Task, option ...OptionI) error
	PublishWithContext(ctx context.Context, task *Task, option ...OptionI) error
	DelayPublish(task *Task, delayTime time.Time, option ...OptionI) error
}

type BeanqSub interface {
	Register(group, queue string, consumerFun DoConsumer)
	StartConsumer()
	StartConsumerWithContext(ctx context.Context)
	StartPing() error
}
