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
	"math/rand"
	"time"

	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/retail-ai-inc/beanq/helper/timex"
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
		Channel() string
		Topic() string
		filter(message *Message) error
	}
	// IBaseSubscribeCmd BaseSubscribeCmd subscribe method
	IBaseSubscribeCmd interface {
		IBaseCmd
		init(broker IBroker) *Subscribe
		Run(ctx context.Context)
	}
	// define available command
	cmdAble func(cmd IBaseCmd) error
	// Client beanq's client
	Client struct {
		broker IBroker

		Topic     string        `json:"topic"`
		Channel   string        `json:"channel"`
		MaxLen    int64         `json:"maxLen"`
		Retry     int           `json:"retry"`
		Priority  float64       `json:"priority"`
		TimeToRun time.Duration `json:"timeToRun"`
	}
)

func New(config *BeanqConfig) *Client {
	// init config,Will merge default options
	config.init()

	client := &Client{
		Topic:     config.Topic,
		Channel:   config.Channel,
		MaxLen:    config.MaxLen,
		Retry:     config.JobMaxRetries,
		Priority:  config.Priority,
		TimeToRun: config.TimeToRun,
	}
	client.broker = NewBroker(config)
	return client
}

func (c *Client) BQ() *BQClient {
	bqc := &BQClient{
		client: &Client{
			broker:    c.broker,
			Topic:     c.Topic,
			Channel:   c.Channel,
			MaxLen:    c.MaxLen,
			Retry:     c.Retry,
			Priority:  c.Priority,
			TimeToRun: c.TimeToRun,
		},

		ctx:      context.Background(),
		id:       "",
		priority: c.Priority,
	}
	bqc.cmdAble = bqc.process
	return bqc
}

func (c *Client) Wait(ctx context.Context) {
	c.broker.startConsuming(ctx)
}

func (c *Client) WaitingAck(ctx context.Context, id string) (msg *Message, err error) {
	pollIntervalBase := time.Millisecond
	maxInterval := 500 * time.Millisecond
	nextPollInterval := func() time.Duration {
		// Add 10% jitter.
		interval := pollIntervalBase + time.Duration(rand.Intn(int(pollIntervalBase/10)))
		// Double and clamp for next time.
		pollIntervalBase *= 2
		if pollIntervalBase > maxInterval {
			pollIntervalBase = maxInterval
		}
		return interval
	}

	var pullAcknowledgement = func(ctx context.Context, id string) (*Message, error) {
		return nil, nil
	}

	timer := time.NewTimer(nextPollInterval())
	defer timer.Stop()
	for {
		if msg, err := pullAcknowledgement(ctx, id); err != nil {
			return msg, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			// pull the data from global
			timer.Reset(nextPollInterval())
		}
	}
}

// Ping this method can be called by user for checking the status of broker
func (c *Client) Ping() {
}

type BQClient struct {
	cmdAble
	client *Client

	ctx context.Context

	// TODO
	// id and priority are not common parameters for all publish and subscription, and will need to be optimized in the future.
	id       string
	priority float64
}

func (b *BQClient) WithContext(ctx context.Context) *BQClient {
	b.ctx = ctx
	return b
}

func (b *BQClient) SetId(id string) *BQClient {
	b.id = id
	return b
}

func (b *BQClient) GetId() string {
	return b.id
}

func (b *BQClient) Priority(priority float64) *BQClient {
	if priority >= 1000 {
		priority = 999
	}
	b.priority = priority
	return b
}

func (b *BQClient) process(cmd IBaseCmd) error {
	var channel, topic = cmd.Channel(), cmd.Topic()
	if channel == "" {
		channel = b.client.Channel
	}
	if topic == "" {
		topic = b.client.Topic
	}

	if cmd, ok := cmd.(*Publish); ok {
		// make message
		message := &Message{
			Topic:       topic,
			Channel:     channel,
			Payload:     stringx.ByteToString(cmd.payload),
			MoodType:    string(cmd.moodType),
			AddTime:     cmd.executeTime.Format(timex.DateTime),
			ExecuteTime: cmd.executeTime,

			Id:       b.id,
			Priority: b.priority,

			MaxLen:       b.client.MaxLen,
			Retry:        b.client.Retry,
			PendingRetry: 0,
			TimeToRun:    b.client.TimeToRun,
		}

		if err := cmd.filter(message); err != nil {
			return err
		}

		if b.id != message.Id {
			b.id = message.Id
		}
		// store message
		return b.client.broker.enqueue(b.ctx, message)
	}
	if cmd, ok := cmd.(*Subscribe); ok {
		b.client.broker.addConsumer(cmd.subscribeType, channel, topic, cmd.handle)
	}
	return nil
}

func (t cmdAble) Publish(channel, topic string, payload []byte) error {
	cmd := &Publish{
		channel:     channel,
		topic:       topic,
		payload:     payload,
		executeTime: time.Now(),
		moodType:    NORMAL,
	}

	if err := t(cmd); err != nil {
		return err
	}
	return nil
}

func (t cmdAble) PublishAtTime(channel, topic string, payload []byte, atTime time.Time) error {
	cmd := &Publish{
		channel:     channel,
		topic:       topic,
		payload:     payload,
		executeTime: atTime,
		moodType:    DELAY,
	}

	if err := t(cmd); err != nil {
		return err
	}
	return nil
}

func (t cmdAble) PublishInSequential(channel, topic string, payload []byte) error {
	cmd := &Publish{
		channel:  channel,
		topic:    topic,
		payload:  payload,
		moodType: SEQUENTIAL,
	}
	if err := t(cmd); err != nil {
		return err
	}
	return nil
}

func (t cmdAble) Subscribe(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	cmd := &Subscribe{
		channel:       channel,
		topic:         topic,
		moodType:      NORMAL,
		handle:        handle,
		subscribeType: normalSubscribe,
	}
	if err := t(cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (t cmdAble) SubscribeDelay(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	cmd := &Subscribe{
		channel:       channel,
		topic:         topic,
		moodType:      DELAY,
		handle:        handle,
		subscribeType: normalSubscribe,
	}
	if err := t(cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (t cmdAble) SubscribeSequential(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	cmd := &Subscribe{
		channel:       channel,
		topic:         topic,
		moodType:      SEQUENTIAL,
		handle:        handle,
		subscribeType: sequentialSubscribe,
	}
	if err := t(cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

type (
	// Publish command:publish
	Publish struct {
		channel, topic string
		payload        []byte
		moodType       moodType
		executeTime    time.Time

		isUnique bool
	}
	// Subscribe command:subscribe
	Subscribe struct {
		channel, topic string
		moodType       moodType

		subscribeType subscribeType
		broker        IBroker
		handle        IConsumeHandle
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

func (t *Publish) Channel() string {
	return t.channel
}

func (t *Publish) Topic() string {
	return t.topic
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

func (t *Subscribe) Channel() string {
	return t.channel
}

func (t *Subscribe) Topic() string {
	return t.topic
}
