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
	"time"

	"github.com/go-redis/redis/v8"
)

type (
	OptionType int

	Option struct {
		ExecuteTime time.Time
		Topic       string
		Channel     string
		OrderKey    string
		Priority    float64
		Retry       int
		MaxLen      int64
	}

	OptionI interface {
		String() string
		OptType() OptionType
		Value() any
	}

	priorityOption float64
	retryOption    int
	topicOption    string
	orderKeyOption string
	channelOption  string
	maxLenOption   int64
	executeTime    time.Time
)

const (
	MaxRetryOpt OptionType = iota + 1
	PriorityOpt
	TopicOpt
	ChannelOpt
	MaxLenOpt
	ExecuteTimeOpt
	IdleTime
	OrderKeyOpt
)

func WithTopic(name string) OptionI {
	return topicOption(name)
}

func (t topicOption) String() string {
	return "queueOption"
}

func (t topicOption) OptType() OptionType {
	return TopicOpt
}

func (t topicOption) Value() any {
	return string(t)
}

func WithRetry(retries int) OptionI {
	if retries < 0 {
		retries = 0
	}
	return retryOption(retries)
}

func (retry retryOption) String() string {
	return "retryOption"
}

func (retry retryOption) OptType() OptionType {
	return MaxRetryOpt
}

func (retry retryOption) Value() any {
	return int(retry)
}

func WithChannel(name string) OptionI {
	return channelOption(name)
}

func (t channelOption) String() string {
	return "channelOption"
}

func (t channelOption) OptType() OptionType {
	return ChannelOpt
}

func (t channelOption) Value() any {
	return string(t)
}

func WithMaxLen(maxLen int) OptionI {
	if maxLen < 0 {
		maxLen = 1000
	}
	return maxLenOption(maxLen)
}

func (ml maxLenOption) String() string {
	return "maxLenOption"
}

func (ml maxLenOption) OptType() OptionType {
	return MaxLenOpt
}

func (ml maxLenOption) Value() any {
	return int(ml)
}

func WithExecuteTime(unixTime time.Time) OptionI {
	if unixTime.IsZero() {
		unixTime = time.Now()
	}
	return executeTime(unixTime)
}

func (et executeTime) String() string {
	return "executeTime"
}

func (et executeTime) OptType() OptionType {
	return ExecuteTimeOpt
}

func (et executeTime) Value() any {
	return time.Time(et)
}

func WithPriority(priority float64) OptionI {
	if priority > 1000 {
		priority = 999
	}
	if priority < 0 {
		priority = 0
	}
	return priorityOption(priority)
}

func (pri priorityOption) String() string {
	return "priorityOption"
}

func (pri priorityOption) OptType() OptionType {
	return PriorityOpt
}

func (pri priorityOption) Value() any {
	return float64(pri)
}

func WithOrderKey(name string) OptionI {
	return orderKeyOption(name)
}

func (t orderKeyOption) String() string {
	return "orderKeyOption"
}

func (t orderKeyOption) OptType() OptionType {
	return OrderKeyOpt
}

func (t orderKeyOption) Value() any {
	return string(t)
}

func ComposeOptions(options ...OptionI) (Option, error) {
	res := Option{
		Priority:    DefaultOptions.Priority,
		Retry:       DefaultOptions.JobMaxRetry,
		Topic:       DefaultOptions.DefaultTopic,
		Channel:     DefaultOptions.DefaultChannel,
		MaxLen:      DefaultOptions.DefaultMaxLen,
		ExecuteTime: time.Now(),
		OrderKey:    DefaultOptions.OrderKey,
	}
	for _, f := range options {
		switch f.OptType() {
		case PriorityOpt:
			if v, ok := f.Value().(float64); ok {
				res.Priority = v
			}
		case TopicOpt:
			if v, ok := f.Value().(string); ok {
				res.Topic = v
			}
		case ChannelOpt:
			if v, ok := f.Value().(string); ok {
				res.Channel = v
			}
		case MaxRetryOpt:
			if v, ok := f.Value().(int); ok {
				res.Retry = v
			}
		case MaxLenOpt:
			if v, ok := f.Value().(int64); ok {
				res.MaxLen = v
			}
		case ExecuteTimeOpt:
			if v, ok := f.Value().(time.Time); ok {
				res.ExecuteTime = v
			}
		case OrderKeyOpt:
			if v, ok := f.Value().(string); ok {
				res.OrderKey = v
			}
		}
	}
	return res, nil
}

type Result struct {
	Id   string
	Args []any
}

type Options struct {
	WorkCount                chan struct{}
	RedisOptions             *redis.Options
	DefaultTopic             string
	DefaultDelayChannel      string
	DefaultDelayTopic        string
	OrderKey                 string
	DefaultChannel           string
	Prefix                   string
	ConsumerPoolSize         int
	Priority                 float64
	JobMaxRetry              int
	DefaultMaxLen            int64
	MinConsumers             int64
	TimeToRun                time.Duration
	KeepSuccessJobsInHistory time.Duration
	KeepFailedJobsInHistory  time.Duration
	RetryTime                time.Duration
	PublishTimeOut           time.Duration
	ConsumeTimeOut           time.Duration
}

var DefaultOptions = &Options{

	KeepFailedJobsInHistory:  time.Hour * 24 * 7,
	KeepSuccessJobsInHistory: time.Hour * 24 * 7,
	PublishTimeOut:           10 * time.Second,
	ConsumeTimeOut:           20 * time.Second,
	ConsumerPoolSize:         20,
	MinConsumers:             10,
	TimeToRun:                3600 * time.Second,
	JobMaxRetry:              3,
	Prefix:                   "beanq",

	Priority:       0,
	DefaultTopic:   "default-topic",
	DefaultChannel: "default-channel",
	DefaultMaxLen:  2000,

	OrderKey: "",

	DefaultDelayTopic:   "default-delay-topic",
	DefaultDelayChannel: "default-delay-channel",

	RetryTime: 800 * time.Millisecond,

	WorkCount: make(chan struct{}, 20),
}
