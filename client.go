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
	"sync/atomic"
	"time"

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
type moodType string

const (
	NORMAL     moodType = "normal"
	DELAY      moodType = "delay"
	SEQUENTIAL moodType = "sequential"
)

type (
	// IBaseCmd BaseCmd public method
	IBaseCmd interface {
		filter(message *Message) error
	}
	// IBaseSubscribeCmd BaseSubscribeCmd subscribe method
	IBaseSubscribeCmd interface {
		IBaseCmd
		init(broker IBroker) *Subscribe
		Run(ctx context.Context)
	}
	// define available command
	cmdAble func(ctx context.Context, cmd IBaseCmd) error
	// Client beanq's client
	Client struct {
		cmdAble
		broker IBroker
		config *BeanqConfig
	}
)

var idAtomic, channelAtomic, topicAtomic, moodTypeAtomic, priorityAtomic, payloadAtomic, executeTimeAtomic, timeToRunAtomic, maxLenAtomic atomic.Value

func New(config *BeanqConfig) *Client {
	// init config,Will merge default options
	config.init()

	client := &Client{}
	idAtomic.Store("")
	channelAtomic.Store(config.Channel)
	topicAtomic.Store(config.Topic)
	moodTypeAtomic.Store(NORMAL)
	priorityAtomic.Store(float64(0))
	payloadAtomic.Store("")
	timeToRunAtomic.Store(config.TimeToRun)
	maxLenAtomic.Store(config.MaxLen)

	client.config = config
	client.cmdAble = client.process
	client.broker = NewBroker(config)
	return client
}

func (t *Client) SetId(idStr string) *Client {
	idAtomic.Store(idStr)
	return t
}

func (t *Client) Channel(name string) *Client {
	channelAtomic.Store(name)
	return t
}

func (t *Client) Topic(name string) *Client {
	topicAtomic.Store(name)
	return t
}

func (t *Client) MoodType(typeName string) *Client {
	moodTypeAtomic.Store(typeName)
	return t
}

func (t *Client) Priority(priorityVal float64) *Client {
	if priorityVal >= 1000 {
		priorityVal = 999
	}
	priorityAtomic.Store(priorityVal)
	return t
}

func (t *Client) TimeToRun(duration time.Duration) *Client {
	timeToRunAtomic.Store(duration)
	return t
}

func (t *Client) MaxLen(maxLen int64) *Client {
	maxLenAtomic.Store(maxLen)
	return t
}

func (t *Client) Payload(payloadVal []byte) *Client {
	payloadAtomic.Store(stringx.ByteToString(payloadVal))
	return t
}

func (t *Client) process(ctx context.Context, cmd IBaseCmd) error {

	msg := &Message{
		Id:        idAtomic.Load().(string),
		Topic:     topicAtomic.Load().(string),
		Channel:   channelAtomic.Load().(string),
		Priority:  priorityAtomic.Load().(float64),
		Payload:   payloadAtomic.Load().(string),
		TimeToRun: timeToRunAtomic.Load().(time.Duration),
		MaxLen:    maxLenAtomic.Load().(int64),
	}
	// reset to default
	idAtomic.Store("")
	topicAtomic.Store(t.config.Topic)
	channelAtomic.Store(t.config.Channel)
	priorityAtomic.Store(float64(0))
	payloadAtomic.Store("")
	moodTypeAtomic.Store(NORMAL)
	executeTimeAtomic.Store(time.Now())
	timeToRunAtomic.Store(t.config.TimeToRun)
	maxLenAtomic.Store(t.config.MaxLen)

	if err := cmd.filter(msg); err != nil {
		return err
	}

	if cmd, ok := cmd.(*Publish); ok {

		msg.ExecuteTime = cmd.executeTime
		msg.MoodType = string(cmd.moodType)

		// store message
		return t.broker.enqueue(ctx, msg)
	}
	if cmd, ok := cmd.(*Subscribe); ok {
		t.broker.addConsumer(cmd.subscribeType, msg.Channel, msg.Topic, cmd.handle)
	}
	return nil
}

func (t *Client) Wait(ctx context.Context) {
	t.broker.startConsuming(ctx)
}

func (t *Client) ping() {

}

func (t cmdAble) Publish(ctx context.Context) error {
	cmd := &Publish{
		moodType:    NORMAL,
		executeTime: time.Now(),
	}
	if err := t(ctx, cmd); err != nil {
		return err
	}
	return nil
}

func (t cmdAble) PublishAtTime(ctx context.Context, atTime time.Time) error {
	cmd := &Publish{
		moodType:    DELAY,
		executeTime: atTime,
	}
	if err := t(ctx, cmd); err != nil {
		return err
	}
	return nil
}

func (t cmdAble) PublishInSequential(ctx context.Context) error {
	cmd := &Publish{
		moodType: SEQUENTIAL,
	}
	if err := t(ctx, cmd); err != nil {
		return err
	}
	return nil
}

func (t cmdAble) Subscribe(ctx context.Context, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	cmd := &Subscribe{
		moodType:      NORMAL,
		handle:        handle,
		subscribeType: normalSubscribe,
	}
	if err := t(ctx, cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (t cmdAble) SubscribeDelay(ctx context.Context, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	cmd := &Subscribe{
		moodType:      DELAY,
		handle:        handle,
		subscribeType: normalSubscribe,
	}
	if err := t(ctx, cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (t cmdAble) SubscribeSequential(ctx context.Context, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	cmd := &Subscribe{
		moodType:      SEQUENTIAL,
		handle:        handle,
		subscribeType: sequentialSubscribe,
	}
	if err := t(ctx, cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

type (
	// Publish command:publish
	Publish struct {
		moodType    moodType
		executeTime time.Time
		isUnique    bool
		IBaseCmd
	}
	// Subscribe command:subscribe
	Subscribe struct {
		moodType moodType
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
