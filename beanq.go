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
	Queue struct {
		Topic        string
		DelayChannel string
		DelayTopic   string
		Channel      string
		MaxLen       int64
		Priority     float64
		TimeToRun    time.Duration
	}
	History struct {
		On    bool
		Mongo struct {
			Database              string
			Collection            string
			UserName              string
			Password              string
			Host                  string
			Port                  string
			ConnectTimeOut        time.Duration
			MaxConnectionPoolSize uint64
			MaxConnectionLifeTime time.Duration
		}
	}
	BeanqConfig struct {
		Health                   Health        `json:"health"`
		DebugLog                 DebugLog      `json:"debugLog"`
		Broker                   string        `json:"broker"`
		Redis                    Redis         `json:"redis"`
		ConsumerPoolSize         int           `json:"consumerPoolSize"`
		JobMaxRetries            int           `json:"jobMaxRetries"`
		DeadLetterIdleTime       time.Duration `json:"deadLetterIdle"`
		DeadLetterTicker         time.Duration `json:"deadLetterTicker"`
		KeepFailedJobsInHistory  time.Duration `json:"keepFailedJobsInHistory"`
		KeepSuccessJobsInHistory time.Duration `json:"keepSuccessJobsInHistory"`
		PublishTimeOut           time.Duration `json:"publishTimeOut"`
		ConsumeTimeOut           time.Duration `json:"consumeTimeOut"`
		MinConsumers             int64         `json:"minConsumers"`
		Queue
		History
	}
)

func (t *BeanqConfig) init() {
	if t.ConsumerPoolSize == 0 {
		t.ConsumerPoolSize = DefaultOptions.ConsumerPoolSize
	}
	if t.JobMaxRetries < 0 {
		t.JobMaxRetries = DefaultOptions.JobMaxRetry
	}
	if t.DeadLetterIdleTime == 0 {
		t.DeadLetterIdleTime = DefaultOptions.DeadLetterIdle
	}
	if t.DeadLetterTicker == 0 {
		t.DeadLetterTicker = DefaultOptions.DeadLetterTicker
	}

	if t.KeepSuccessJobsInHistory == 0 {
		t.KeepSuccessJobsInHistory = DefaultOptions.KeepSuccessJobsInHistory
	}
	if t.KeepFailedJobsInHistory == 0 {
		t.KeepFailedJobsInHistory = DefaultOptions.KeepFailedJobsInHistory
	}
	if t.PublishTimeOut == 0 {
		t.PublishTimeOut = DefaultOptions.PublishTimeOut
	}
	if t.ConsumeTimeOut == 0 {
		t.ConsumeTimeOut = DefaultOptions.ConsumeTimeOut
	}
	if t.MinConsumers == 0 {
		t.MinConsumers = DefaultOptions.MinConsumers
	}
	if t.Channel == "" {
		t.Channel = DefaultOptions.DefaultChannel
	}
	if t.Topic == "" {
		t.Topic = DefaultOptions.DefaultTopic
	}
	if t.DelayChannel == "" {
		t.DelayChannel = DefaultOptions.DefaultDelayChannel
	}
	if t.DelayTopic == "" {
		t.DelayTopic = DefaultOptions.DefaultDelayTopic
	}
	if t.MaxLen == 0 {
		t.MaxLen = DefaultOptions.DefaultMaxLen
	}
	if t.TimeToRun == 0 {
		t.TimeToRun = DefaultOptions.TimeToRun
	}
	if t.History.Mongo.Collection == "" {
		t.History.Mongo.Collection = "event_logs"
	}
	if t.History.Mongo.ConnectTimeOut == 0 {
		t.History.Mongo.ConnectTimeOut = 10 * time.Second
	}
	if t.History.Mongo.MaxConnectionPoolSize == 0 {
		t.History.Mongo.MaxConnectionPoolSize = 200
	}
	if t.History.Mongo.MaxConnectionLifeTime == 0 {
		t.History.Mongo.MaxConnectionLifeTime = 600 * time.Second
	}
}

// IHandle consumer ,after broker
type IHandle interface {
	Channel() string
	Topic() string
	Process(ctx context.Context)
	Schedule(ctx context.Context) error
	DeadLetter(ctx context.Context) error
}

// VolatileLFU ...
type VolatileLFU interface {
	Add(ctx context.Context, key, member string) (bool, error)
	Delete(ctx context.Context, key string) error
}
