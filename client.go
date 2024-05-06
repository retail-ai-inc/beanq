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
	"fmt"
	"time"

	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/rs/xid"
)

// subscribe type
type subscribeType int

const (
	normalSubscribe subscribeType = iota + 1
	sequentialSubscribe
)

// message type
const (
	NORMAL     = "normal"
	DELAY      = "delay"
	SEQUENTIAL = "sequential"
)

type (
	// BaseCmd public method
	IBaseCmd interface {
		filter(message *Message) error
	}
	// BaseSubscribeCmd subscribe method
	IBaseSubscribeCmd interface {
		IBaseCmd
		init(broker IBroker) *Subscribe
		Run(ctx context.Context)
	}
	// define available command
	cmdAble func(ctx context.Context, cmd IBaseCmd) error
	// Client beanq's client
	Client struct {
		message *Message
		cmdAble
		broker IBroker
	}
)

func New(config *BeanqConfig) *Client {
	// init config,Will merge default options
	config.init()

	client := &Client{}
	now := time.Now()
	client.message = &Message{
		TopicName:   config.Topic,
		ChannelName: config.Channel,
		MaxLen:      config.MaxLen,
		Retry:       config.JobMaxRetries,
		Priority:    config.Priority,
		AddTime:     now.Format("2006-01-02 15:04:05"),
		ExecuteTime: now,
		TimeToRun:   config.TimeToRun,
	}
	client.cmdAble = client.process
	client.broker = NewBroker(config)
	return client
}

func (t *Client) SetId(id string) *Client {
	t.message.Id = id
	return t
}

func (t *Client) Channel(name string) *Client {
	t.message.ChannelName = name
	return t
}

func (t *Client) Topic(name string) *Client {
	t.message.TopicName = name
	return t
}

func (t *Client) MoodType(typeName string) *Client {
	t.message.MoodType = typeName
	return t
}

func (t *Client) Priority(priority float64) *Client {
	if priority > 1000 {
		priority = 999
	}
	t.message.Priority = priority
	return t
}

func (t *Client) Payload(payload []byte) *Client {
	t.message.Payload = stringx.ByteToString(payload)
	return t
}

func (t *Client) process(ctx context.Context, cmd IBaseCmd) error {

	if err := cmd.filter(t.message); err != nil {
		return err
	}

	if cmd, ok := cmd.(*Publish); ok {
		t.message.ExecuteTime = cmd.executeTime
		t.message.MoodType = cmd.moodType
		return t.broker.enqueue(ctx, t.message)
	}
	if cmd, ok := cmd.(*Subscribe); ok {
		t.broker.addConsumer(cmd.subscribeType, t.message.ChannelName, t.message.TopicName, cmd.handle)
	}
	return nil
}

func (t *Client) Wait(ctx context.Context) {
	t.broker.startConsuming(ctx)
}

func (t *Client) ping() {

}

func (t cmdAble) Publish(ctx context.Context) {
	cmd := &Publish{
		moodType:    NORMAL,
		executeTime: time.Now(),
	}
	if err := t(ctx, cmd); err != nil {
		logger.New().Error(err)
	}
	return
}

func (t cmdAble) PublishAtTime(ctx context.Context, atTime time.Time) {
	cmd := &Publish{
		moodType:    DELAY,
		executeTime: atTime,
	}
	if err := t(ctx, cmd); err != nil {
		logger.New().Error(err)
	}
	return
}

func (t cmdAble) PublishInSequential(ctx context.Context) {
	cmd := &Publish{
		moodType: SEQUENTIAL,
	}
	if err := t(ctx, cmd); err != nil {
		logger.New().Error(err)
	}
	return
}

func (t cmdAble) Subscribe(ctx context.Context, handle IConsumeHandle) IBaseSubscribeCmd {
	cmd := &Subscribe{
		moodType:      NORMAL,
		handle:        handle,
		subscribeType: normalSubscribe,
	}
	if err := t(ctx, cmd); err != nil {
		logger.New().Error(err)
		return nil
	}
	return cmd
}

func (t cmdAble) SubscribeDelay(ctx context.Context, handle IConsumeHandle) IBaseSubscribeCmd {
	cmd := &Subscribe{
		moodType:      DELAY,
		handle:        handle,
		subscribeType: normalSubscribe,
	}
	if err := t(ctx, cmd); err != nil {
		logger.New().Error(err)
		return nil
	}
	return cmd
}

func (t cmdAble) SubscribeSequential(ctx context.Context, handle IConsumeHandle) IBaseSubscribeCmd {
	cmd := &Subscribe{
		moodType:      SEQUENTIAL,
		handle:        handle,
		subscribeType: sequentialSubscribe,
	}
	if err := t(ctx, cmd); err != nil {
		logger.New().Error(err)
		return nil
	}
	return cmd
}

type (
	// Publish command:publish
	Publish struct {
		moodType    string
		executeTime time.Time
		isUnique    bool
		IBaseCmd
	}
	// Subscribe command:subscribe
	Subscribe struct {
		moodType string
		subscribeType
		IBaseSubscribeCmd
		broker IBroker
		handle IConsumeHandle
	}
)

func (t *Publish) filter(message *Message) error {

	if message.Id == "" {
		guid := xid.NewWithTime(time.Now())
		message.Id = guid.String()
	}

	if message.Payload == "" {
		return errors.New("missing Payload")
	}
	return nil
}

func (t *Subscribe) filter(message *Message) error {
	return nil
}

func (t *Subscribe) init(broker IBroker) *Subscribe {
	t.broker = broker

	return t
}

// Run will to be implemented
func (t *Subscribe) Run(ctx context.Context) {
	fmt.Println("will implement")
}
