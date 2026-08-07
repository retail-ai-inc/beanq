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
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/rs/xid"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/helper/timex"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/boptions"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/bredis"
)

var (
	brokerDriverMu sync.RWMutex
	brokerDriver   any
)

var (
	_ Broker          = (*bredis.Broker)(nil)
	_ MigrationRunner = (*bredis.Broker)(nil)
	_ adminReporter   = (*bredis.Broker)(nil)
	_ driverProvider  = (*bredis.Broker)(nil)
)

type Handler struct {
	do        public.CallbackWithRetry
	channel   string
	topic     string
	moodType  btype.MoodType
	retryCond map[string]struct{}
}

type consumerRegistry struct {
	mu       sync.Mutex
	handlers []*Handler
}

const clientShutdownCleanupTimeout = 5 * time.Second

func (r *consumerRegistry) add(handler *Handler) {
	r.mu.Lock()
	r.handlers = append(r.handlers, handler)
	r.mu.Unlock()
}

func (r *consumerRegistry) drain() []*Handler {
	r.mu.Lock()
	handlers := r.handlers
	r.handlers = nil
	r.mu.Unlock()
	return handlers
}

func (h *Handler) Invoke(ctx context.Context, broker Broker) error {
	callback := public.NewCallbackWithRetry(func(ctx context.Context, data map[string]any, retry ...int) (int, error) {
		attempt, err := h.do.Handle(ctx, data, retry...)
		if err != nil {
			if _, ok := h.retryCond[retryConditionKey(err)]; ok {
				return attempt, noRetryError{err: err}
			}
		}
		return attempt, err
	}, h.do.Error)
	return broker.Consume(ctx, h.moodType, h.channel, h.topic, callback)
}

type noRetryError struct {
	err error
}

func (e noRetryError) Error() string            { return e.err.Error() }
func (e noRetryError) Unwrap() error            { return e.err }
func (e noRetryError) BeanqRetryStopped() error { return e.err }

var (
	ErrNilHandle = errors.New("beanq:handle is nil")
	ErrNilCancel = errors.New("beanq:cancel is nil")
)

type (
	IConsumeHandle interface {
		Handle(ctx context.Context, message *Message) error
	}

	IConsumeCancel interface {
		Cancel(ctx context.Context, message *Message) error
	}

	IConsumeError interface {
		Error(ctx context.Context, err error)
	}

	DefaultHandle struct {
		DoHandle func(ctx context.Context, message *Message) error
		DoCancel func(ctx context.Context, message *Message) error
		DoError  func(ctx context.Context, err error)
	}
	WorkflowHandler func(ctx context.Context, wf *Workflow) error
)

func (c WorkflowHandler) Handle(ctx context.Context, message *Message) error {
	workflow, err := NewWorkflow(ctx, message)
	if err != nil {
		return err
	}
	return c(ctx, workflow)
}

func (c DefaultHandle) Handle(ctx context.Context, message *Message) error {
	if c.DoHandle != nil {
		return c.DoHandle(ctx, message)
	}
	return ErrNilHandle
}

func (c DefaultHandle) Cancel(ctx context.Context, message *Message) error {
	if c.DoCancel != nil {
		return c.DoCancel(ctx, message)
	}
	return ErrNilCancel
}

func (c DefaultHandle) Error(ctx context.Context, err error) {
	if c.DoError != nil {
		c.DoError(ctx, err)
	}
}

var MigrateLogDiscard MigrationRunner = discard{}

type discard struct{}

func (discard) Migrate(context.Context, []map[string]any) error {
	return nil
}

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
		closeOnce        sync.Once
		closeErr         error
		captureException func(ctx context.Context, err any)
		broker           Broker
		driver           any
		consumers        *consumerRegistry
		captureConfig    *capture.Config
		TimeToRunLimit   []time.Duration `json:"timeToRunLimit"`
		Topic            string          `json:"topic"`
		Channel          string          `json:"channel"`
		MaxLen           int64           `json:"maxLen"`
		Retry            int             `json:"retry"`
		DeadLetterRetry  int             `json:"deadLetterRetry"`
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

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		if closer, ok := c.broker.(interface{ Close() error }); ok {
			c.closeErr = closer.Close()
		}
	})
	return c.closeErr
}

func New(config *BeanqConfig, options ...ClientOption) *Client {
	if config == nil {
		logger.New().Panic("new client err:", berror.ErrInvalidConfig.WithMessage("config is nil"))
	}
	candidate := ResolvedConfig{BeanqConfig: cloneBeanqConfig(*config)}
	candidate.ApplyDefaults()

	client := newClientFromConfig(&candidate.BeanqConfig)
	for _, option := range options {
		option(client)
	}
	if client.broker == nil {
		resolved, err := config.Resolve()
		if err != nil {
			logger.New().Panic("new client err:", err)
		}
		candidate = resolved
		client.broker, client.captureConfig = newBrokerFromConfig(candidate)
	}
	setBrokerDriver(client.broker)
	if provider, ok := client.broker.(driverProvider); ok {
		client.driver = provider.Driver()
	}
	client.config = &candidate.BeanqConfig
	return client
}

func newBrokerFromConfig(config ResolvedConfig) (Broker, *capture.Config) {
	switch config.Broker {
	case "redis":
		return newRedisBroker(config)
	default:
		logger.New().Panic("new broker err:", berror.ErrUnsupportedBroker.WithMessage(config.Broker))
		return nil, nil
	}
}

func newRedisBroker(config ResolvedConfig) (Broker, *capture.Config) {
	driver, err := bredis.NewRedisClient(context.Background(), config.redisClientOptions())
	if err != nil {
		logger.New().Panic("new broker err:", err)
	}

	broker := bredis.NewBrokerWithOptions(driver, config.redisBrokerOptions())

	var captureConfig *capture.Config
	var migrator MigrationRunner
	if config.History.On && config.Mongo != nil {
		captureConfig = loadCaptureConfig(config.Mongo)
		store, err := newMongoStore(context.Background(), config.Mongo)
		if err != nil {
			logger.New().Panic("new mongo store err:", err)
		}
		migrator = store
	}
	broker.Configure(captureConfig, migrator)
	return broker, captureConfig
}

type captureConfigReader interface {
	ConfigInfo(ctx context.Context) (*capture.Config, error)
}

func loadCaptureConfig(config *Mongo) *capture.Config {
	store := bmongo.NewMongo(config.Host, config.Port, config.UserName, config.Password, config.Database, config.collectionNames(),
		config.ConnectTimeOut, config.MaxConnectionPoolSize, config.MaxConnectionLifeTime,
		bmongo.MongoSSLConfig{On: config.SSL.On, CAFile: config.SSL.CAFile, Verify: config.SSL.Verify})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	captureConfig, err := captureConfigReader(store).ConfigInfo(ctx)
	if err != nil {
		return nil
	}
	return captureConfig
}

func setBrokerDriver(broker Broker) {
	provider, ok := broker.(driverProvider)
	if !ok {
		return
	}
	brokerDriverMu.Lock()
	brokerDriver = provider.Driver()
	brokerDriverMu.Unlock()
}

func GetBrokerDriver[T any]() T {
	brokerDriverMu.RLock()
	driver := brokerDriver
	brokerDriverMu.RUnlock()
	if driver == nil {
		logger.New().Panic("the broker has not been initialized yet")
	}
	typedDriver, ok := driver.(T)
	if !ok {
		logger.New().Panic("broker driver has unexpected type")
	}
	return typedDriver
}

func newClientFromConfig(config *BeanqConfig) *Client {
	return &Client{
		consumers:       &consumerRegistry{},
		Topic:           config.Topic,
		Channel:         config.Channel,
		MaxLen:          config.MaxLen,
		Retry:           config.JobMaxRetries,
		DeadLetterRetry: 0,
		Priority:        config.Priority,
		TimeToRun:       config.TimeToRun,
	}
}

type sequenceAckWaiter interface {
	WaitingSequenceAck(ctx context.Context, channel, topic, orderKey, id string) (map[string]string, error)
}

func (c *Client) WaitingAck(ctx context.Context, channel, topic, id string) (*Message, error) {
	data, err := c.broker.WaitingAck(ctx, channel, topic, id)
	if err != nil {
		return nil, err
	}
	return MessageS(data).ToMessage(), nil
}

func (c *Client) WaitingSequenceAck(ctx context.Context, channel, topic, orderKey, id string) (*Message, error) {
	if waiter, ok := c.broker.(sequenceAckWaiter); ok {
		data, err := waiter.WaitingSequenceAck(ctx, channel, topic, orderKey, id)
		if err != nil {
			return nil, err
		}
		return MessageS(data).ToMessage(), nil
	}
	return c.WaitingAck(ctx, channel, topic, id)
}

func WithBroker(broker Broker) ClientOption {
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
		driver:           c.driver,
		consumers:        c.consumers,
		captureConfig:    c.captureConfig,
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
	defer func() {
		if err := c.Close(); err != nil {
			logger.New().Error(err)
		}
	}()
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	handlersDone := c.startHandlers(ctx)
	c.startMigration(ctx)
	c.startHostReporter(ctx)

	logger.New().Info("Beanq Start")
	<-ctx.Done()
	select {
	case <-handlersDone:
	case <-time.After(c.gracefulShutdownTimeout() + clientShutdownCleanupTimeout):
		logger.New().Warn("Beanq graceful shutdown timed out")
	}
	logger.New().Info("Beanq Stop")
	_ = logger.New().Sync()
}

func (c *Client) gracefulShutdownTimeout() time.Duration {
	if c.config != nil && c.config.GracefulShutdownTimeout > 0 {
		return c.config.GracefulShutdownTimeout
	}
	return boptions.DefaultGracefulShutdownTimeout
}

func (c *Client) startHandlers(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	var wait sync.WaitGroup
	if c.consumers == nil {
		close(done)
		return done
	}
	for _, handler := range c.consumers.drain() {
		if handler == nil {
			continue
		}
		hdl := *handler
		wait.Go(func() {
			if err := hdl.Invoke(ctx, c.broker); err != nil {
				hdl.do.Error(ctx, err)
			}
		})
	}
	go func() {
		wait.Wait()
		close(done)
	}()
	return done
}

func (c *Client) startMigration(ctx context.Context) {
	migrator, ok := c.broker.(MigrationRunner)
	if !ok {
		return
	}
	go func() {
		if ctx.Err() != nil {
			return
		}
		if err := migrator.Migrate(ctx, nil); err != nil {
			panic(err)
		}
	}()
}

func (c *Client) startHostReporter(ctx context.Context) {
	admin, ok := c.broker.(adminReporter)
	if !ok {
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
				if err := admin.HostName(ctx); err != nil {
					fmt.Printf("hostname err:%+v \n", err)
				}
			}
		}
	}()
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
	if c.consumers == nil {
		c.consumers = &consumerRegistry{}
	}
	c.consumers.add(&handler)
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

	h, ok := subscribe.(IConsumeCancel)
	if !ok {
		return err
	}
	cancelErr := h.Cancel(ctx, msg)
	if cancelErr == nil || errors.Is(cancelErr, ErrNilCancel) {
		return err
	}
	return errors.Join(err, cancelErr)

}

func (c *Client) CheckAckStatus(ctx context.Context, channel, topic, id string) (*Message, error) {
	return c.WaitingAck(ctx, channel, topic, id)
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

func (b *BQClient) PublishSequence(channel, topic, orderKey string, payload []byte) *SequenceCmd {
	return b.publishSequence(Publish{
		channel:     channel,
		topic:       topic,
		payload:     payload,
		orderKey:    orderKey,
		moodType:    btype.SEQUENCE_QUEUE,
		executeTime: time.Now(),
	})
}

func (b *BQClient) publishSequence(cmd Publish) *SequenceCmd {
	channel, topic := b.client.resolveChannelTopic(cmd.channel, cmd.topic)
	sequenceCmd := &SequenceCmd{
		channel:  channel,
		topic:    topic,
		orderKey: cmd.orderKey,
		ctx:      b.ctx,
		client:   b.client,
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
	if cmd.moodType == btype.SEQUENCE_QUEUE && cmd.orderKey == "" {
		return errors.New("please configure orderKey")
	}
	return nil
}

func (b *BQClient) buildMessage(cmd *Publish) *Message {
	channel, topic := b.client.resolveChannelTopic(cmd.channel, cmd.topic)
	messageID := b.id
	if messageID == "" {
		messageID = xid.New().String()
	}
	return &Message{
		Topic:           topic,
		Channel:         channel,
		OrderKey:        cmd.orderKey,
		Payload:         string(cmd.payload),
		MoodType:        cmd.moodType,
		AddTime:         cmd.executeTime.Format(timex.DateTime),
		ExecuteTime:     cmd.executeTime,
		Id:              messageID,
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

func (t cmdAble) SubscribeSequence(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	return t.subscribe(channel, topic, btype.SEQUENCE_QUEUE, btype.SequentialSubscribe, handle)
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
		executeTime time.Time
		channel     string
		topic       string
		orderKey    string
		moodType    btype.MoodType
		payload     []byte
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
	err      error
	ctx      context.Context
	client   *Client
	channel  string
	topic    string
	orderKey string
	id       string
}

func (s *SequenceCmd) Error() error {
	return s.err
}

// WaitingAck waits for and returns the acknowledgement status.
func (s *SequenceCmd) WaitingAck() (*Message, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.orderKey != "" {
		return s.client.WaitingSequenceAck(s.ctx, s.channel, s.topic, s.orderKey, s.id)
	}
	return s.client.WaitingAck(s.ctx, s.channel, s.topic, s.id)
}

func DynamicKeyOpt(key string) DynamicOption {
	return func(option *dynamicOption) {
		option.key = key
	}
}
