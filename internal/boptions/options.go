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

package boptions

import (
	"time"
)

type Result struct {
	Id   string
	Args []any
}

type Options struct {
	WorkCount                chan struct{}
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
	DeadLetterIdle           time.Duration
	DeadLetterTicker         time.Duration
	TimeToRun                time.Duration
	KeepSuccessJobsInHistory time.Duration
	KeepFailedJobsInHistory  time.Duration
	RetryTime                time.Duration
	PublishTimeOut           time.Duration
	ConsumeTimeOut           time.Duration
}

var DefaultOptions = &Options{
	DeadLetterIdle:           time.Second * 60,
	DeadLetterTicker:         time.Second * 5,
	KeepFailedJobsInHistory:  time.Hour * 24 * 7,
	KeepSuccessJobsInHistory: time.Hour * 24 * 7,
	PublishTimeOut:           10 * time.Second,
	ConsumeTimeOut:           20 * time.Second,
	ConsumerPoolSize:         10,
	MinConsumers:             100,
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
