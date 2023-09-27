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

	"github.com/redis/go-redis/v9"
)

type (
	OptionType int

	Option struct {
		Priority    float64
		Retry       int
		Queue       string
		Group       string
		MaxLen      int64
		ExecuteTime time.Time
	}

	OptionI interface {
		String() string
		OptType() OptionType
		Value() any
	}

	priorityOption float64
	retryOption    int
	queueOption    string
	groupOption    string
	maxLenOption   int64
	executeTime    time.Time
)

const (
	MaxRetryOpt OptionType = iota + 1
	PriorityOpt
	QueueOpt
	GroupOpt
	MaxLenOpt
	ExecuteTimeOpt
	IdleTime
)

func Queue(name string) OptionI {
	return queueOption(name)
}

func (queue queueOption) String() string {
	return "queueOption"
}

func (queue queueOption) OptType() OptionType {
	return QueueOpt
}

func (queue queueOption) Value() any {
	return string(queue)
}

func Retry(retries int) OptionI {
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

func Group(name string) OptionI {
	return groupOption(name)
}

func (group groupOption) String() string {
	return "groupOption"
}

func (group groupOption) OptType() OptionType {
	return GroupOpt
}

func (group groupOption) Value() any {
	return string(group)
}

func MaxLen(maxLen int) OptionI {
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

func ExecuteTime(unixTime time.Time) OptionI {
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

func Priority(priority float64) OptionI {
	if priority > 10 {
		priority = 10
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

func ComposeOptions(options ...OptionI) (Option, error) {
	res := Option{
		Priority:    DefaultOptions.Priority,
		Retry:       DefaultOptions.JobMaxRetry,
		Queue:       DefaultOptions.DefaultQueueName,
		Group:       DefaultOptions.DefaultGroup,
		MaxLen:      DefaultOptions.DefaultMaxLen,
		ExecuteTime: time.Now(),
	}
	for _, f := range options {
		switch f.OptType() {
		case PriorityOpt:
			if v, ok := f.Value().(float64); ok {
				res.Priority = v
			}
		case QueueOpt:
			if v, ok := f.Value().(string); ok {
				res.Queue = v
			}
		case GroupOpt:
			if v, ok := f.Value().(string); ok {
				res.Group = v
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
		}
	}
	return res, nil
}

// TODO: need more parameters
type Result struct {
	Id   string
	Args []any
}

type Options struct {
	RedisOptions *redis.Options

	KeepJobInQueue           time.Duration
	KeepFailedJobsInHistory  time.Duration
	KeepSuccessJobsInHistory time.Duration

	PoolSize    int
	MinWorkers  int
	JobMaxRetry int
	Prefix      string
	Priority    float64

	DefaultQueueName, DefaultGroup string
	DefaultMaxLen                  int64

	DefaultDelayQueueName, DefaultDelayGroup string

	RetryTime time.Duration
	WorkCount chan struct{}
}

var DefaultOptions = &Options{

	KeepJobInQueue:           time.Hour * 24 * 7,
	KeepFailedJobsInHistory:  time.Hour * 24 * 7,
	KeepSuccessJobsInHistory: time.Hour * 24 * 7,
	PoolSize:                 20,
	MinWorkers:               10,
	JobMaxRetry:              3,
	Prefix:                   "beanq",

	Priority:         0,
	DefaultQueueName: "default-queue",
	DefaultGroup:     "default-group",
	DefaultMaxLen:    2000,

	DefaultDelayQueueName: "default-delay-queue",
	DefaultDelayGroup:     "default-delay-group",

	RetryTime: 800 * time.Millisecond,
	WorkCount: make(chan struct{}, 20),
}
