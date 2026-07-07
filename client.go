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
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/helper/timex"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/rs/xid"
)

type (
	// IBaseCmd is the base command contract used by publish and subscribe commands.
	IBaseCmd interface {
		filter(message *Message) error
	}

	// IBaseSubscribeCmd is returned by subscribe calls.
	IBaseSubscribeCmd interface {
		IBaseCmd
		Run(ctx context.Context)
	}

	cmdAble func(cmd IBaseCmd) error

	// Client is BeanQ's root client.
	Client struct {
		captureException func(ctx context.Context, err any)
		broker           *Broker
		TimeToRunLimit   []time.Duration `json:"timeToRunLimit"`
		Topic            string          `json:"topic"`
		Channel          string          `json:"channel"`
		MaxLen           int64           `json:"maxLen"`
		Retry            int             `json:"retry"`
		DeadLetterRetry  int             `json:"deadletterRetry"`
		Priority         float64         `json:"priority"`
		TimeToRun        time.Duration   `json:"timeToRun"`
		retryConditions  []RetryConditionFunc
		config           *BeanqConfig
	}

	dynamicOption struct {
		key string
		on  bool
	}

	DynamicOption func(option *dynamicOption)
	ClientOption  func(client *Client)

	// RetryConditionFunc decides whether a handler error should be retried.
	RetryConditionFunc func(map[string]any, error) bool
)

func New(config *BeanqConfig, options ...ClientOption) *Client {
	config.init()

	client := newClientFromConfig(config)
	for _, option := range options {
		option(client)
	}
	if client.broker == nil {
		client.broker = NewBroker(config)
	}
	client.config = config
	return client
}

func newClientFromConfig(config *BeanqConfig) *Client {
	return &Client{
		Topic:           config.Topic,
		Channel:         config.Channel,
		MaxLen:          config.MaxLen,
		Retry:           config.JobMaxRetries,
		DeadLetterRetry: 0,
		Priority:        config.Priority,
		TimeToRun:       config.TimeToRun,
	}
}

// ForceUnlock force deletes an order key.
func (c *Client) ForceUnlock(ctx context.Context, channel, topic, orderKey string) error {
	return c.broker.ForceUnlock(ctx, channel, topic, orderKey)
}

func WithBroker(broker *Broker) ClientOption {
	return func(client *Client) {
		client.broker = broker
	}
}

func WithCaptureExceptionOption(handler func(ctx context.Context, err any)) ClientOption {
	return func(client *Client) {
		client.captureException = handler
	}
}

func WithRetryConditions(condition ...RetryConditionFunc) ClientOption {
	return func(client *Client) {
		client.retryConditions = append(client.retryConditions, condition...)
	}
}

func (c *Client) BQ() *BQClient {
	bqc := &BQClient{
		client:        c.cloneForCommand(),
		dynamicOption: &dynamicOption{},
		ctx:           context.Background(),
		priority:      c.Priority,
	}
	bqc.cmdAble = bqc.process
	return bqc
}

func (c *Client) cloneForCommand() *Client {
	return &Client{
		broker:           c.broker,
		Topic:            c.Topic,
		Channel:          c.Channel,
		MaxLen:           c.MaxLen,
		Retry:            c.Retry,
		DeadLetterRetry:  c.DeadLetterRetry,
		Priority:         c.Priority,
		TimeToRun:        c.TimeToRun,
		TimeToRunLimit:   slices.Clone(c.TimeToRunLimit),
		captureException: c.captureException,
		retryConditions:  slices.Clone(c.retryConditions),
		config:           c.config,
	}
}

func (c *Client) Wait(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	c.startHandlers(ctx)
	c.startMigration(ctx)
	c.startHostReporter(ctx)

	logger.New().Info("Beanq Start")
	<-c.WaitSignal(cancel)
}

func (c *Client) startHandlers(ctx context.Context) {
	for key, handler := range c.broker.handlers {
		if handler == nil {
			continue
		}
		hdl := *handler
		go func() {
			brokerImpl := c.broker.Mood(hdl.moodType)
			hdl.Invoke(ctx, brokerImpl)
		}()
		c.broker.handlers[key] = nil
	}
}

func (c *Client) startMigration(ctx context.Context) {
	go func() {
		if ctx.Err() != nil {
			return
		}
		if err := c.broker.Migrate(ctx, nil); err != nil {
			panic(err)
		}
	}()
}

func (c *Client) startHostReporter(ctx context.Context) {
	if c.broker == nil || c.broker.tool == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.broker.tool.HostName(ctx); err != nil {
					fmt.Printf("hostname err:%+v \n", err)
				}
			}
		}
	}()
}

func (c *Client) WaitSignal(cancel context.CancelFunc) <-chan bool {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigs
		cancel()
		_ = logger.New().Sync()
		done <- true
	}()
	return done
}

func (c *Client) AddConsumer(moodType btype.MoodType, channel, topic string, subscribe IConsumeHandle, retryConditions map[string]struct{}) error {
	if subscribe == nil {
		return ErrNilHandle
	}

	handler := Handler{
		channel:   channel,
		topic:     topic,
		moodType:  moodType,
		retryCond: retryConditions,
		do:        consumerCallback(subscribe),
	}
	c.broker.handlers = append(c.broker.handlers, &handler)
	return nil
}

func consumerCallback(subscribe IConsumeHandle) public.CallbackWithRetry {
	return public.NewCallbackWithRetry(func(ctx context.Context, message map[string]any, retry ...int) (int, error) {
		msg := messageToStruct(message)
		if err := subscribe.Handle(ctx, msg); err != nil {
			return 0, consumeCancel(ctx, subscribe, msg, err)
		}
		return 0, nil
	}, func(ctx context.Context, err error) {
		if h, ok := subscribe.(IConsumeError); ok {
			h.Error(ctx, err)
		}
	})
}

func consumeCancel(ctx context.Context, subscribe IConsumeHandle, msg *Message, err error) error {
	var joined error
	joined = errors.Join(joined, err)
	if h, ok := subscribe.(IConsumeCancel); ok {
		joined = errors.Join(joined, h.Cancel(ctx, msg))
	}
	return joined
}

func (c *Client) CheckAckStatus(ctx context.Context, channel, topic, id string, isOrder bool) (*Message, error) {
	m, err := c.broker.Status(ctx, channel, topic, id, isOrder)
	if err != nil {
		return nil, err
	}
	return MessageS(m).ToMessage(), nil
}

// Ping can be called by users to check the broker status.
func (c *Client) Ping() {}

type BQClient struct {
	ctx context.Context
	cmdAble
	client          *Client
	dynamicOption   *dynamicOption
	id              string
	priority        float64
	waitAck         bool
	lockOrderKeyTTL time.Duration
	retryConditions map[string]struct{}
}

func (b *BQClient) WithContext(ctx context.Context) *BQClient {
	if ctx == nil {
		ctx = context.Background()
	}
	b.ctx = ctx
	return b
}

// Dynamic only supports Sequential type for now.
func (b *BQClient) Dynamic(options ...DynamicOption) *BQClient {
	opt := &dynamicOption{on: true}
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

// SetTimeToRun sets the consumer execution timeout and optional alert thresholds.
func (b *BQClient) SetTimeToRun(duration time.Duration, limits ...time.Duration) *BQClient {
	if duration > 0 {
		b.client.TimeToRun = duration
		b.client.TimeToRunLimit = limits
	}
	return b
}

// Retry dynamically sets the consumption retry count.
func (b *BQClient) Retry(retry int) *BQClient {
	if retry <= 0 {
		retry = 0
	}
	b.client.Retry = retry
	return b
}

// SetLockOrderKeyTTL sets the sequence-by-lock key TTL. Values <= 0 never expire unless force-unlocked.
func (b *BQClient) SetLockOrderKeyTTL(duration time.Duration) *BQClient {
	b.lockOrderKeyTTL = duration
	return b
}

func (b *BQClient) IgnoreRetryConditions(errs ...error) *BQClient {
	retryConditions := make(map[string]struct{}, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		retryConditions[retryConditionKey(err)] = struct{}{}
	}
	b.retryConditions = retryConditions
	return b
}

func retryConditionKey(err error) string {
	return fmt.Sprintf("%T,%v", err, err.Error())
}

func (b *BQClient) PublishInSequence(channel, topic string, payload []byte) *SequenceCmd {
	return b.publishSequence(Publish{
		channel:     channel,
		topic:       topic,
		payload:     payload,
		moodType:    btype.SEQUENCE,
		executeTime: time.Now(),
	}, false)
}

func (b *BQClient) PublishInSequenceByLock(channel, topic, orderKey string, payload []byte) *SequenceCmd {
	return b.publishSequence(Publish{
		channel:         channel,
		topic:           topic,
		payload:         payload,
		orderKey:        orderKey,
		lockOrderKeyTTL: b.lockOrderKeyTTL,
		moodType:        btype.SEQUENCE_BY_LOCK,
		executeTime:     time.Now(),
	}, true)
}

func (b *BQClient) publishSequence(cmd Publish, isOrder bool) *SequenceCmd {
	channel, topic := b.client.resolveChannelTopic(cmd.channel, cmd.topic)
	sequenceCmd := &SequenceCmd{
		channel: channel,
		topic:   topic,
		ctx:     b.ctx,
		client:  b.client,
		isOrder: isOrder,
	}
	if err := b.process(&cmd); err != nil {
		sequenceCmd.err = err
	} else {
		sequenceCmd.id = b.id
	}
	return sequenceCmd
}

func (b *BQClient) process(cmd IBaseCmd) error {
	switch cmd := cmd.(type) {
	case *Publish:
		return b.processPublish(cmd)
	case *Subscribe:
		return b.processSubscribe(cmd)
	default:
		return fmt.Errorf("unknown structure type: %T", cmd)
	}
}

func (b *BQClient) processPublish(cmd *Publish) error {
	if err := b.validatePublish(cmd); err != nil {
		return err
	}

	message := b.buildMessage(cmd)
	if err := cmd.filter(message); err != nil {
		return err
	}
	b.id = message.Id
	return b.client.broker.Enqueue(b.ctx, message.ToMap())
}

func (b *BQClient) validatePublish(cmd *Publish) error {
	b.waitAck = cmd.moodType == btype.SEQUENCE
	if cmd.moodType == btype.SEQUENCE && b.id == "" {
		return errors.New("please configure a unique ID")
	}
	return nil
}

func (b *BQClient) buildMessage(cmd *Publish) *Message {
	channel, topic := b.client.resolveChannelTopic(cmd.channel, cmd.topic)
	return &Message{
		Topic:           topic,
		Channel:         channel,
		OrderKey:        cmd.orderKey,
		LockOrderKeyTTL: cmd.lockOrderKeyTTL,
		Payload:         string(cmd.payload),
		MoodType:        cmd.moodType,
		AddTime:         cmd.executeTime.Format(timex.DateTime),
		ExecuteTime:     cmd.executeTime,
		Id:              b.id,
		Priority:        b.priority,
		MaxLen:          b.client.MaxLen,
		Retry:           b.client.Retry,
		DeadLetterRetry: b.client.DeadLetterRetry,
		PendingRetry:    0,
		TimeToRun:       b.client.TimeToRun,
		TimeToRunLimit:  b.client.TimeToRunLimit,
	}
}

func (b *BQClient) processSubscribe(cmd *Subscribe) error {
	if b.dynamicOption.on {
		// Reserved for future dynamic subscription support.
		return nil
	}
	channel, topic := b.client.resolveChannelTopic(cmd.channel, cmd.topic)
	return b.client.AddConsumer(cmd.moodType, channel, topic, cmd.handle, b.retryConditions)
}

func (c *Client) resolveChannelTopic(channel, topic string) (string, string) {
	if channel == "" {
		channel = c.Channel
	}
	if topic == "" {
		topic = c.Topic
	}
	return channel, topic
}

func (t cmdAble) Publish(channel, topic string, payload []byte) error {
	return t(&Publish{
		channel:     channel,
		topic:       topic,
		payload:     payload,
		executeTime: time.Now(),
		moodType:    btype.NORMAL,
	})
}

func (t cmdAble) PublishAtTime(channel, topic string, payload []byte, atTime time.Time) error {
	return t(&Publish{
		channel:     channel,
		topic:       topic,
		payload:     payload,
		executeTime: atTime,
		moodType:    btype.DELAY,
	})
}

func (t cmdAble) Subscribe(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	return t.subscribe(channel, topic, btype.NORMAL, btype.NormalSubscribe, handle)
}

func (t cmdAble) SubscribeToDelay(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	return t.subscribe(channel, topic, btype.DELAY, btype.NormalSubscribe, handle)
}

func (t cmdAble) SubscribeToSequence(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	return t.subscribe(channel, topic, btype.SEQUENCE, btype.SequentialSubscribe, handle)
}

func (t cmdAble) SubscribeToSequenceByLock(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	return t.subscribe(channel, topic, btype.SEQUENCE_BY_LOCK, btype.SequentialByLockSubscribe, handle)
}

func (t cmdAble) subscribe(channel, topic string, moodType btype.MoodType, subscribeType btype.SubscribeType, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	cmd := &Subscribe{
		channel:       channel,
		topic:         topic,
		moodType:      moodType,
		handle:        handle,
		subscribeType: subscribeType,
	}
	if err := t(cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

type (
	// Publish command.
	Publish struct {
		executeTime     time.Time
		channel         string
		topic           string
		orderKey        string
		lockOrderKeyTTL time.Duration
		moodType        btype.MoodType
		payload         []byte
	}

	// Subscribe command.
	Subscribe struct {
		handle        IConsumeHandle
		channel       string
		topic         string
		moodType      btype.MoodType
		subscribeType btype.SubscribeType
	}
)

func (t *Publish) filter(message *Message) error {
	if message.Id == "" {
		message.Id = xid.NewWithTime(time.Now()).String()
	}
	if message.Payload == "" {
		return errors.New("missing Payload")
	}
	return nil
}

func (t *Subscribe) filter(message *Message) error {
	return nil
}

// Run will be implemented in a future release.
func (t *Subscribe) Run(ctx context.Context) {
	fmt.Println("will implement")
}

type SequenceCmd struct {
	err     error
	ctx     context.Context
	client  *Client
	channel string
	topic   string
	id      string
	isOrder bool
}

func (s *SequenceCmd) Error() error {
	return s.err
}

// WaitingAck waits for and returns the acknowledgement status.
func (s *SequenceCmd) WaitingAck() (*Message, error) {
	if s.err != nil {
		return nil, s.err
	}
	nack, err := s.client.broker.Status(s.ctx, s.channel, s.topic, s.id, s.isOrder)
	if err != nil {
		return nil, err
	}
	return MessageS(nack).ToMessage(), nil
}

func DynamicKeyOpt(key string) DynamicOption {
	return func(option *dynamicOption) {
		option.key = key
	}
}
