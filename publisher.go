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
	"errors"
	"sync"
	"time"
)

type (
	MoodType int

	PubClient struct {
		broker         Broker
		wg             *sync.WaitGroup
		publishTimeOut time.Duration
		channelName    string
		topicName      string
		maxLen         int64
		retry          int
		priority       float64
		mood           MoodType
	}
)

const (
	_ MoodType = iota
	NORMAL
	DELAY
	SEQUENTIAL
)

var _ BeanqPub = (*PubClient)(nil)

func NewPublisher(config BeanqConfig) *PubClient {
	opts := DefaultOptions

	poolSize := config.ConsumerPoolSize
	if poolSize <= 0 {
		poolSize = DefaultOptions.ConsumerPoolSize
	}

	publishTimeOut := config.PublishTimeOut
	if publishTimeOut <= 0 {
		publishTimeOut = opts.PublishTimeOut
		config.PublishTimeOut = opts.PublishTimeOut
	}

	Config.Store(config)

	return &PubClient{
		broker:         NewBroker(config),
		wg:             nil,
		publishTimeOut: publishTimeOut,
		channelName:    DefaultOptions.DefaultChannel,
		topicName:      DefaultOptions.DefaultTopic,
		maxLen:         DefaultOptions.DefaultMaxLen,
		retry:          DefaultOptions.JobMaxRetry,
		priority:       DefaultOptions.Priority,
	}
}

func (t *PubClient) Channel(name string) *PubClient {
	if name != "" {
		t.channelName = name
	}
	return t
}

func (t *PubClient) Topic(name string) *PubClient {
	if name != "" {
		t.topicName = name
	}
	return t
}

func (t *PubClient) MaxLen(maxLen int64) *PubClient {
	if maxLen > 0 {
		t.maxLen = maxLen
	}
	return t
}

func (t *PubClient) Retry(retry int) *PubClient {
	if retry > 0 {
		t.retry = retry
	}
	return t
}

func (t *PubClient) Priority(priority float64) *PubClient {
	if priority > 1000 {
		t.priority = 999
	}
	if priority <= 0 {
		t.priority = 0
	}
	return t
}

func (t *PubClient) reset() {
	t.channelName = DefaultOptions.DefaultChannel
	t.topicName = DefaultOptions.DefaultTopic
	t.retry = DefaultOptions.JobMaxRetry
	t.priority = DefaultOptions.Priority
	t.maxLen = DefaultOptions.DefaultMaxLen
}

func (t *PubClient) PublishWithContext(ctx context.Context, msg *Message, option ...OptionI) error {
	defer func() {
		t.reset()
	}()

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.publishTimeOut)
		defer cancel()
	}

	opts, err := ComposeOptions(option...)
	if err != nil {
		return err
	}
	msg.TopicName = t.topicName
	msg.ChannelName = t.channelName
	msg.Retry = t.retry
	msg.Priority = t.priority
	msg.MaxLen = t.maxLen
	msg.ExecuteTime = opts.ExecuteTime
	msg.MoodType = "normal"

	if opts.ExecuteTime.After(time.Now()) {
		msg.MoodType = "delay"
	}
	if opts.OrderKey != "" {
		msg.MoodType = "sequential"
	}

	return t.broker.enqueue(ctx, msg, opts)
}

func (t *PubClient) PublishAtTime(msg *Message, delay time.Time, option ...OptionI) error {
	msg.MoodType = "delay"
	option = append(option, ExecuteTime(delay))
	return t.Publish(msg, option...)
}

func (t *PubClient) PublishWithDelay(msg *Message, delay time.Duration, option ...OptionI) error {
	msg.MoodType = "delay"
	delayTime := time.Now().Add(delay)
	option = append(option, ExecuteTime(delayTime))
	return t.Publish(msg, option...)
}

func (t *PubClient) PublishInSequence(msg *Message, orderKey string, option ...OptionI) error {
	msg.MoodType = "sequential"
	if orderKey == "" {
		return errors.New("orderKey can't be empty")
	}
	option = append(option, OrderKey(orderKey))
	return t.Publish(msg, option...)
}

func (t *PubClient) Publish(msg *Message, option ...OptionI) error {
	msg.MoodType = "normal"
	ctx, cancel := context.WithTimeout(context.Background(), t.publishTimeOut)
	defer cancel()
	return t.PublishWithContext(ctx, msg, option...)
}

func (t *PubClient) Close() error {
	return t.broker.close()
}
