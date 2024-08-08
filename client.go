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
	"flag"
	"fmt"
	"time"

	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/retail-ai-inc/beanq/helper/timex"
	"github.com/rs/xid"
)

// subscribe type
type subscribeType int

const (
	normalSubscribe     = subscribeType(1)
	sequentialSubscribe = subscribeType(2)
)

// MoodType message type
type MoodType string

func (m MoodType) String() string {
	return fmt.Sprintf("mood type: %s", string(m))
}

func (m MoodType) MarshalBinary() ([]byte, error) {
	return []byte(m), nil
}

const (
	NORMAL     MoodType = "normal"
	DELAY      MoodType = "delay"
	SEQUENTIAL MoodType = "sequential"
)

type (
	// IBaseCmd BaseCmd public method
	IBaseCmd interface {
		filter(message *Message) error
	}
	// IBaseSubscribeCmd BaseSubscribeCmd subscribe method
	IBaseSubscribeCmd interface {
		IBaseCmd
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

		captureException func(ctx context.Context, err any)
	}

	dynamicOption struct {
		on  bool
		key string
	}

	DynamicOption func(option *dynamicOption)
	ClientOption  func(client *Client)
)

var (
	on       = flag.Bool("on", false, "mongo log enable")
	database = flag.String("database", "", "Mongo database name for saving logs")
	username = flag.String("username", "", "Mongo username")
	password = flag.String("password", "", "Mongo password")
	host     = flag.String("host", "", "Mongo host")
	port     = flag.String("port", "", "Mongo port")
)

func New(config *BeanqConfig, options ...ClientOption) *Client {
	// init config,Will merge default options
	config.init()

	flag.Parse()
	if *on {
		config.History.On = true
	}
	if *database != "" {
		config.History.Mongo.Database = *database
	}
	if *username != "" {
		config.History.Mongo.UserName = *username
	}
	if *password != "" {
		config.History.Mongo.Password = *password
	}
	if *host != "" {
		config.History.Mongo.Host = *host
	}
	if *port != "" {
		config.History.Mongo.Port = *port
	}

	client := &Client{
		Topic:     config.Topic,
		Channel:   config.Channel,
		MaxLen:    config.MaxLen,
		Retry:     config.JobMaxRetries,
		Priority:  config.Priority,
		TimeToRun: config.TimeToRun,
	}

	for _, option := range options {
		option(client)
	}

	broker := NewBroker(config)
	if client.captureException != nil {
		broker.setCaptureException(client.captureException)
	}

	client.broker = broker
	return client
}

func WithCaptureExceptionOption(handler func(ctx context.Context, err any)) ClientOption {
	return func(client *Client) {
		client.captureException = handler
	}
}

func (c *Client) BQ() *BQClient {
	bqc := &BQClient{
		client: &Client{
			broker:           c.broker,
			Topic:            c.Topic,
			Channel:          c.Channel,
			MaxLen:           c.MaxLen,
			Retry:            c.Retry,
			Priority:         c.Priority,
			TimeToRun:        c.TimeToRun,
			captureException: c.captureException,
		},

		dynamicOption: &dynamicOption{},
		ctx:           context.Background(),
		id:            "",
		priority:      c.Priority,
	}
	bqc.cmdAble = bqc.process
	return bqc
}

func (c *Client) Wait(ctx context.Context) {
	c.broker.startConsuming(ctx)
}

func (c *Client) CheckAckStatus(ctx context.Context, channel, id string) (*Message, error) {
	return c.broker.checkStatus(ctx, channel, id)
}

// Ping this method can be called by user for checking the status of broker
func (c *Client) Ping() {
}

type BQClient struct {
	cmdAble
	client *Client

	ctx context.Context

	waitAck       bool
	dynamicOption *dynamicOption

	// TODO
	// id and priority are not common parameters for all publish and subscription, and will need to be optimized in the future.
	id       string
	priority float64
}

func (b *BQClient) WithContext(ctx context.Context) *BQClient {
	b.ctx = ctx
	return b
}

// Dynamic only support Sequential type for now.
func (b *BQClient) Dynamic(options ...DynamicOption) *BQClient {
	opt := &dynamicOption{
		on:  true,
		key: "",
	}
	for _, option := range options {
		option(opt)
	}

	b.dynamicOption = opt
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

func (b *BQClient) PublishInSequential(channel, topic string, payload []byte) *SequentialCmd {
	cmd := &Publish{
		channel:     channel,
		topic:       topic,
		payload:     payload,
		moodType:    SEQUENTIAL,
		executeTime: time.Now(),
	}
	sequentialCmd := &SequentialCmd{
		err:     nil,
		channel: channel,
		topic:   topic,
		ctx:     b.ctx,
		client:  b.client,
	}
	if err := b.process(cmd); err != nil {
		sequentialCmd.err = err
	} else {
		sequentialCmd.id = b.id
	}

	return sequentialCmd
}

func (b *BQClient) process(cmd IBaseCmd) error {
	switch cmd := cmd.(type) {
	case *Publish:
		channel, topic := cmd.channel, cmd.topic

		if channel == "" {
			channel = b.client.Channel
		}
		if topic == "" {
			topic = b.client.Topic
		}

		b.waitAck = cmd.moodType == SEQUENTIAL
		if cmd.moodType == SEQUENTIAL {
			if b.id == "" {
				return errors.New("please configure a unique ID")
			}
		}
		// make message
		message := &Message{
			Topic:       topic,
			Channel:     channel,
			Payload:     stringx.ByteToString(cmd.payload),
			MoodType:    cmd.moodType,
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
		return b.client.broker.enqueue(b.ctx, message, b.dynamicOption.on)

	case *Subscribe:
		channel, topic := cmd.channel, cmd.topic

		if channel == "" {
			channel = b.client.Channel
		}
		if topic == "" {
			topic = b.client.Topic
		}

		if b.dynamicOption.on {
			b.client.broker.dynamicConsuming(cmd.subscribeType, channel, cmd.handle, b.dynamicOption.key)
		} else {
			b.client.broker.addConsumer(cmd.subscribeType, channel, topic, cmd.handle)
		}
	default:
		return fmt.Errorf("unknown structure type: %T", cmd)
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

func (t cmdAble) PPublish(channel, topic string, payload []byte) error {
	cmd := &Publish{
		channel:     channel,
		topic:       topic,
		payload:     payload,
		executeTime: time.Now(),
		moodType:    SEQUENTIAL,
	}

	if err := t(cmd); err != nil {
		return err
	}
	return nil
}

func (t cmdAble) PUnSubscribe() {}

type (
	// Publish command:publish
	Publish struct {
		channel, topic string
		payload        []byte
		moodType       MoodType
		executeTime    time.Time
	}

	// Subscribe command:subscribe
	Subscribe struct {
		channel, topic string
		moodType       MoodType

		subscribeType subscribeType
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

func (t *Subscribe) filter(message *Message) error {
	return nil
}

// Run will to be implemented
func (t *Subscribe) Run(ctx context.Context) {
	fmt.Println("will implement")
}

type SequentialCmd struct {
	err                error
	ctx                context.Context
	channel, topic, id string
	client             *Client
}

func (s *SequentialCmd) Error() error {
	return s.err
}

// WaitingAck ...
func (s *SequentialCmd) WaitingAck(ctx context.Context, id string) (*Message, error) {
	if s.err != nil {
		return nil, s.err
	}
	nack, err := s.client.broker.monitorStream(ctx, s.channel, s.topic, id)
	if err != nil {
		return nil, err
	}
	return nack, nil
}

func DynamicKeyOpt(key string) DynamicOption {
	return func(option *dynamicOption) {
		option.key = key
	}
}
