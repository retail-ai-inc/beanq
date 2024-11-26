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
	"github.com/retail-ai-inc/beanq/v3/internal/boptions"
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
		On bool
	}
	BeanqConfig struct {
		Health   Health   `json:"health"`
		Broker   string   `json:"broker"`
		DebugLog DebugLog `json:"debugLog"`
		History
		Queue
		Redis                    Redis         `json:"redis"`
		DeadLetterIdleTime       time.Duration `json:"deadLetterIdle"`
		DeadLetterTicker         time.Duration `json:"deadLetterTicker"`
		KeepFailedJobsInHistory  time.Duration `json:"keepFailedJobsInHistory"`
		KeepSuccessJobsInHistory time.Duration `json:"keepSuccessJobsInHistory"`
		PublishTimeOut           time.Duration `json:"publishTimeOut"`
		ConsumeTimeOut           time.Duration `json:"consumeTimeOut"`
		MinConsumers             int64         `json:"minConsumers"`
		JobMaxRetries            int           `json:"jobMaxRetries"`
		ConsumerPoolSize         int           `json:"consumerPoolSize"`
	}
)

func (t *BeanqConfig) init() {
	if t.ConsumerPoolSize == 0 {
		t.ConsumerPoolSize = boptions.DefaultOptions.ConsumerPoolSize
	}
	if t.JobMaxRetries < 0 {
		t.JobMaxRetries = boptions.DefaultOptions.JobMaxRetry
	}
	if t.DeadLetterIdleTime == 0 {
		t.DeadLetterIdleTime = boptions.DefaultOptions.DeadLetterIdle
	}
	if t.DeadLetterTicker == 0 {
		t.DeadLetterTicker = boptions.DefaultOptions.DeadLetterTicker
	}

	if t.KeepSuccessJobsInHistory == 0 {
		t.KeepSuccessJobsInHistory = boptions.DefaultOptions.KeepSuccessJobsInHistory
	}
	if t.KeepFailedJobsInHistory == 0 {
		t.KeepFailedJobsInHistory = boptions.DefaultOptions.KeepFailedJobsInHistory
	}
	if t.PublishTimeOut == 0 {
		t.PublishTimeOut = boptions.DefaultOptions.PublishTimeOut
	}
	if t.ConsumeTimeOut == 0 {
		t.ConsumeTimeOut = boptions.DefaultOptions.ConsumeTimeOut
	}
	if t.MinConsumers == 0 {
		t.MinConsumers = boptions.DefaultOptions.MinConsumers
	}
	if t.Channel == "" {
		t.Channel = boptions.DefaultOptions.DefaultChannel
	}
	if t.Topic == "" {
		t.Topic = boptions.DefaultOptions.DefaultTopic
	}
	if t.DelayChannel == "" {
		t.DelayChannel = boptions.DefaultOptions.DefaultDelayChannel
	}
	if t.DelayTopic == "" {
		t.DelayTopic = boptions.DefaultOptions.DefaultDelayTopic
	}
	if t.MaxLen == 0 {
		t.MaxLen = boptions.DefaultOptions.DefaultMaxLen
	}
	if t.TimeToRun == 0 {
		t.TimeToRun = boptions.DefaultOptions.TimeToRun
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
