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
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/helper/timex"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/retail-ai-inc/beanq/v4/internal/routers"
	"github.com/rs/xid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
		captureException func(ctx context.Context, err any)
		broker           *Broker
		TimeToRunLimit   []time.Duration `json:"timeToRunLimit"`
		Topic            string          `json:"topic"`
		Channel          string          `json:"channel"`
		MaxLen           int64           `json:"maxLen"`
		Retry            int             `json:"retry"`
		Priority         float64         `json:"priority"`
		TimeToRun        time.Duration   `json:"timeToRun"`
		retryConditions  []RetryConditionFunc
	}

	dynamicOption struct {
		key string
		on  bool
	}

	DynamicOption func(option *dynamicOption)
	ClientOption  func(client *Client)
	// Include payload details in the method retry condition, as it may be complex.
	RetryConditionFunc func(map[string]any, error) bool
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

	client.broker = NewBroker(config)
	return client
}

// ForceUnlock force delete a order key
func (c *Client) ForceUnlock(ctx context.Context, channel, topic, orderKey string) error {
	return c.broker.ForceUnlock(ctx, channel, topic, orderKey)
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
		client: &Client{
			broker:           c.broker,
			Topic:            c.Topic,
			Channel:          c.Channel,
			MaxLen:           c.MaxLen,
			Retry:            c.Retry,
			Priority:         c.Priority,
			TimeToRun:        c.TimeToRun,
			captureException: c.captureException,
			retryConditions:  slices.Clone(c.retryConditions),
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
	ctx, cancel := context.WithCancel(ctx)
	for key, handler := range c.broker.handlers {
		go func(hdl Handler) {
			brokerImpl := c.broker.Mood(hdl.moodType)
			hdl.Invoke(ctx, brokerImpl)
		}(*handler)
		c.broker.handlers[key] = nil
	}

	go func() {
		err := c.broker.Migrate(ctx, nil)
		if err != nil {
			panic(err)
		}
	}()

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

	logger.New().Info("Beanq Start")
	// monitor signal
	<-c.WaitSignal(cancel)
}

func (t *Client) WaitSignal(cancel context.CancelFunc) <-chan bool {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigs
		cancel()
		// t.asyncPool.Release()
		_ = logger.New().Sync()
		done <- true
	}()
	return done
}

func (t *Client) AddConsumer(moodType btype.MoodType, channel, topic string, subscribe IConsumeHandle, retryConditions ...RetryConditionFunc) error {
	conditions := t.retryConditions
	if len(retryConditions) > 0 {
		conditions = append(conditions, retryConditions...)
	}

	handler := Handler{
		channel:         channel,
		topic:           topic,
		moodType:        moodType,
		retryConditions: conditions,
		do: func(ctx context.Context, message map[string]any, retry ...int) (int, error) {
			var gerr error
			msg := messageToStruct(message)
			if err := subscribe.Handle(ctx, msg); err != nil {
				gerr = errors.Join(gerr, err)
				if h, ok := subscribe.(IConsumeCancel); ok {
					gerr = errors.Join(gerr, h.Cancel(ctx, msg))
				}
			}
			return 0, gerr
		},
	}

	t.broker.handlers = append(t.broker.handlers, &handler)
	return nil
}

func (c *Client) CheckAckStatus(ctx context.Context, channel, topic, id string, isOrder bool) (*Message, error) {

	m, err := c.broker.Status(ctx, channel, topic, id, isOrder)

	if err != nil {
		return nil, err
	}

	return MessageS(m).ToMessage(), nil
}

func getRootPath() (string, error) {

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to get caller information")
	}

	dir := filepath.Dir(filename)
	for {

		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}

		nextDir := filepath.Dir(dir)
		if nextDir == dir {
			break
		}
		dir = nextDir
	}
	return dir, nil
}

func StaticFileInfo() (map[string]time.Time, error) {

	files := make(map[string]time.Time, 0)

	dir, err := getRootPath()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "./ui")
	err = filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if !info.IsDir() {
			arr := strings.SplitAfter(path, "ui")
			if len(arr) == 2 {
				files[arr[1]] = info.ModTime()
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}

//go:embed ui
var views embed.FS

func (c *Client) ServeHttp(ctx context.Context) {

	files, err := StaticFileInfo()
	if err != nil {
		logger.New().Error(err)
		capture.System.When(c.broker.captureConfig).Then(err)
	}

	go func() {
		timer := timex.TimerPool.Get(10 * time.Second)
		defer timer.Stop()

		for range timer.C {
			select {
			case <-ctx.Done():
				return
			default:

			}
			timer.Reset(10 * time.Second)
			if err := c.broker.tool.QueueMessage(ctx); err != nil {
				logger.New().Error(err)
			}
		}
	}()
	// compatible with unmodified env.json
	httpport := strings.TrimLeft(c.broker.config.UI.Port, ":")
	httpport = fmt.Sprintf(":%s", httpport)

	if err := os.Setenv("GODEBUG", "httpmuxgo122=1"); err != nil {
		logger.New().Error("Error setting environment variables")
		capture.System.When(c.broker.captureConfig).Then(err)
	}

	mux := http.NewServeMux()

	history := c.broker.config.History
	var mog *bmongo.BMongo
	if history.On {

		// compatible with unmodified env.json
		mongoPort := strings.TrimLeft(history.Mongo.Port, ":")
		mongoPort = fmt.Sprintf(":%s", mongoPort)

		mog = bmongo.NewMongo(
			history.Mongo.Host,
			mongoPort,
			history.Mongo.UserName,
			history.Mongo.Password,
			history.Mongo.Database,
			history.Mongo.Collections,
			history.Mongo.ConnectTimeOut,
			history.Mongo.MaxConnectionPoolSize,
			history.Mongo.MaxConnectionLifeTime,
		)
	}

	var workflowMongoCollection *mongo.Collection
	workflowRecordCfg := c.broker.config.Workflow.Record
	if workflowRecordCfg.On && workflowRecordCfg.Mongo != nil && workflowRecordCfg.Mongo.Database != "" {
		connURI := "mongodb://" + workflowRecordCfg.Mongo.Host + ":" + workflowRecordCfg.Mongo.Port
		opts := options.Client().
			ApplyURI(connURI).
			SetConnectTimeout(workflowRecordCfg.Mongo.ConnectTimeOut).
			SetMaxPoolSize(workflowRecordCfg.Mongo.MaxConnectionPoolSize).
			SetMaxConnIdleTime(workflowRecordCfg.Mongo.MaxConnectionLifeTime)

		if workflowRecordCfg.Mongo.UserName != "" && workflowRecordCfg.Mongo.Password != "" {
			opts.SetAuth(options.Credential{
				AuthSource: workflowRecordCfg.Mongo.Database,
				Username:   workflowRecordCfg.Mongo.UserName,
				Password:   workflowRecordCfg.Mongo.Password,
			})
		}

		client, err := mongo.Connect(ctx, opts)
		if err != nil {
			panic(err)
		}
		workflowMongoCollection = client.Database(workflowRecordCfg.Mongo.Database).Collection(workflowRecordCfg.Mongo.Collection)
	}

	routers.NewRouters(
		mux,
		views,
		files,
		c.broker.client.(redis.UniversalClient),
		mog, workflowMongoCollection,
		c.broker.config.Redis.Prefix, c.broker.config.UI)

	log.Printf("server start on port %+v", httpport)
	if err := http.ListenAndServe(httpport, mux); err != nil {
		capture.System.When(c.broker.captureConfig).Then(err)
		log.Fatalln(err)
	}
}

// Ping this method can be called by user for checking the status of broker
func (c *Client) Ping() {
}

type BQClient struct {
	ctx context.Context
	cmdAble
	client          *Client
	dynamicOption   *dynamicOption
	id              string
	priority        float64
	waitAck         bool
	lockOrderKeyTTL time.Duration
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

func (b *BQClient) SetTimeToRun(duration time.Duration, limit ...time.Duration) *BQClient {
	if duration > 0 {
		b.client.TimeToRun = duration
		b.client.TimeToRunLimit = limit
	}
	return b
}

// If duration <= 0, it will never expire unless ForceUnlock is used to force deletion.
func (b *BQClient) SetLockOrderKeyTTL(duration time.Duration) *BQClient {

	b.lockOrderKeyTTL = duration
	return b
}

func (b *BQClient) PublishInSequence(channel, topic string, payload []byte) *SequenceCmd {
	cmd := &Publish{
		channel:     channel,
		topic:       topic,
		payload:     payload,
		moodType:    btype.SEQUENCE,
		executeTime: time.Now(),
	}
	sequentialCmd := &SequenceCmd{
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

func (b *BQClient) PublishInSequenceByLock(channel, topic, orderKey string, payload []byte) *SequenceCmd {
	cmd := &Publish{
		channel:         channel,
		topic:           topic,
		payload:         payload,
		orderKey:        orderKey,
		lockOrderKeyTTL: b.lockOrderKeyTTL,
		moodType:        btype.SEQUENCE_BY_LOCK,
		executeTime:     time.Now(),
	}
	sequenceCmd := &SequenceCmd{
		err:     nil,
		channel: channel,
		topic:   topic,
		ctx:     b.ctx,
		client:  b.client,
		isOrder: true,
	}
	if err := b.process(cmd); err != nil {
		sequenceCmd.err = err
	} else {
		sequenceCmd.id = b.id
	}

	return sequenceCmd
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

		b.waitAck = cmd.moodType == btype.SEQUENCE
		if cmd.moodType == btype.SEQUENCE {
			if b.id == "" {
				return errors.New("please configure a unique ID")
			}
		}
		// make message
		message := &Message{
			Topic:           topic,
			Channel:         channel,
			OrderKey:        cmd.orderKey,
			LockOrderKeyTTL: cmd.lockOrderKeyTTL,

			Payload:     string(cmd.payload),
			MoodType:    cmd.moodType,
			AddTime:     cmd.executeTime.Format(timex.DateTime),
			ExecuteTime: cmd.executeTime,

			Id:       b.id,
			Priority: b.priority,

			MaxLen:         b.client.MaxLen,
			Retry:          b.client.Retry,
			PendingRetry:   0,
			TimeToRun:      b.client.TimeToRun,
			TimeToRunLimit: b.client.TimeToRunLimit,
		}

		if err := cmd.filter(message); err != nil {
			return err
		}

		if b.id != message.Id {
			b.id = message.Id
		}

		// store message
		return b.client.broker.Enqueue(b.ctx, message.ToMap())
		// return b.client.broker.enqueue(b.ctx, message, b.dynamicOption.on)

	case *Subscribe:
		channel, topic := cmd.channel, cmd.topic

		if channel == "" {
			channel = b.client.Channel
		}
		if topic == "" {
			topic = b.client.Topic
		}

		if b.dynamicOption.on {
			// TODO: maybe need this feature in the future.
		} else {
			if err := b.client.AddConsumer(cmd.moodType, channel, topic, cmd.handle); err != nil {
				return err
			}
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
		moodType:    btype.NORMAL,
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
		moodType:    btype.DELAY,
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
		moodType:      btype.NORMAL,
		handle:        handle,
		subscribeType: btype.NormalSubscribe,
	}
	if err := t(cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (t cmdAble) SubscribeToDelay(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	cmd := &Subscribe{
		channel:       channel,
		topic:         topic,
		moodType:      btype.DELAY,
		handle:        handle,
		subscribeType: btype.NormalSubscribe,
	}
	if err := t(cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (t cmdAble) SubscribeToSequence(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	cmd := &Subscribe{
		channel:       channel,
		topic:         topic,
		moodType:      btype.SEQUENCE,
		handle:        handle,
		subscribeType: btype.SequentialSubscribe,
	}
	if err := t(cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (t cmdAble) SubscribeToSequenceByLock(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error) {
	cmd := &Subscribe{
		channel:       channel,
		topic:         topic,
		moodType:      btype.SEQUENCE_BY_LOCK,
		handle:        handle,
		subscribeType: btype.SequentialByLockSubscribe,
	}
	if err := t(cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

type (
	// Publish command:publish
	Publish struct {
		executeTime     time.Time
		channel         string
		topic           string
		orderKey        string
		lockOrderKeyTTL time.Duration
		moodType        btype.MoodType
		payload         []byte
	}

	// Subscribe command:subscribe
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

// WaitingAck ...
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
