package beanq

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/internal"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/retail-ai-inc/beanq/v3/internal/driver/bredis"
	"github.com/spf13/cast"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/retail-ai-inc/beanq/v3/helper/logger"
)

type (
	IBroker interface {
		driver() any

		enqueue(ctx context.Context, msg *Message, dynamicOn bool) error
		addConsumer(subscribeType btype.SubscribeType, channel, topic string, subscribe IConsumeHandle) *RedisHandle
		addDynamicConsumer(subType btype.SubscribeType, channel, topic string, subscribe IConsumeHandle, streamKey, dynamicKey string) *RedisHandle
		dynamicConsuming(subType btype.SubscribeType, channel string, subscribe IConsumeHandle, dynamicKey string)

		monitorStream(ctx context.Context, channel, id string) (*Message, error)
		setCaptureException(fn func(ctx context.Context, err any))
	}
)

var (
	brokerOnce sync.Once
	broker     Broker
	rdb        redis.UniversalClient
	rdbOnce    sync.Once
)

type Handler struct {
	brokerImpl public.IBroker
	do         func(ctx context.Context, data map[string]any) error
	broker     string
	prefix     string
	channel    string
	topic      string
	moodType   btype.MoodType
}

type Broker struct {
	config   *BeanqConfig
	status   public.IStatus
	log      public.IProcessLog
	handlers []Handler
}

func NewRedisClient(config *BeanqConfig) redis.UniversalClient {

	rdbOnce.Do(func() {
		ctx := context.Background()

		hosts := strings.Split(config.Redis.Host, ",")
		for i, h := range hosts {
			hs := strings.Split(h, ":")
			if len(hs) == 1 {
				hosts[i] = strings.Join([]string{h, config.Redis.Port}, ":")
			}
		}

		rdb = redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:        hosts,
			Password:     config.Redis.Password,
			DB:           config.Redis.Database,
			MaxRetries:   config.Redis.MaxRetries,
			DialTimeout:  config.Redis.DialTimeout,
			ReadTimeout:  config.Redis.ReadTimeout,
			WriteTimeout: config.Redis.WriteTimeout,
			PoolSize:     config.Redis.PoolSize,
			MinIdleConns: config.Redis.MinIdleConnections,
			PoolTimeout:  config.Redis.PoolTimeout,
			PoolFIFO:     false,
		})

		if err := rdb.Ping(ctx).Err(); err != nil {
			logger.New().Fatal(err.Error())
		}
	})

	return rdb
}

func NewBroker(config *BeanqConfig) *Broker {

	brokerOnce.Do(func() {
		switch config.Broker {
		case "redis":

			client := NewRedisClient(config)

			broker.status = bredis.NewStatus(client, config.Redis.Prefix)
			broker.log = bredis.NewProcessLog(client, config.Redis.Prefix)

		default:
			logger.New().Panic("not support broker type:", config.Broker)
		}

	})
	broker.config = config
	return &broker
}

func (t *Broker) Enqueue(ctx context.Context, data map[string]any) error {

	//todo panic
	defer func() {
		if err := recover(); err != nil {

		}
	}()

	moodType := btype.NORMAL

	if v, ok := data["moodType"]; ok {
		moodType = btype.MoodType(cast.ToString(v))
	}

	var bk public.IBroker
	// redis broker

	if t.config.Broker == "redis" {
		bk = bredis.SwitchBroker(NewRedisClient(t.config), t.config.Redis.Prefix, t.config.MaxLen, moodType)
	}

	if err := bk.Enqueue(ctx, data); err != nil {
		return err
	}
	data["status"] = bstatus.StatusPublished

	if err := t.log.AddLog(ctx, data); err != nil {
		return err
	}

	return nil
}

func (t *Broker) Dequeue(ctx context.Context, channel, topic string, do public.CallBack) {

}

func (t *Broker) Status(ctx context.Context, channel, id string) (map[string]string, error) {

	data, err := t.status.Status(ctx, channel, id)
	if err != nil {
		// todo
		return nil, err
	}
	return data, nil
}

func (t *Broker) AddConsumer(moodType btype.MoodType, channel, topic string, subscribe IConsumeHandle) error {

	handler := Handler{
		broker:   t.config.Broker,
		prefix:   t.config.Redis.Prefix,
		channel:  channel,
		topic:    topic,
		moodType: moodType,
		do: func(ctx context.Context, message map[string]any) error {
			return subscribe.Handle(ctx, messageToStruct(message))
		},
	}
	if t.config.Broker == "redis" {
		handler.brokerImpl = bredis.SwitchBroker(NewRedisClient(t.config), t.config.Redis.Prefix, t.config.MaxLen, moodType)
	}
	t.handlers = append(t.handlers, handler)

	return nil
}

func (t *Broker) Start(ctx context.Context) {

	ctx, cancel := context.WithCancel(ctx)

	for _, handler := range t.handlers {
		hdl := handler
		go func(hdl2 Handler) {
			hdl2.brokerImpl.Dequeue(ctx, hdl2.channel, hdl2.topic, hdl2.do)
		}(hdl)
	}
	//dead letter
	//move logs from redis to mongo

	logger.New().Info("Beanq Start")
	// monitor signal
	<-t.WaitSignal(cancel)
}

func (t *Broker) WaitSignal(cancel context.CancelFunc) <-chan bool {

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigs
		cancel()
		//t.asyncPool.Release()
		_ = logger.New().Sync()
		done <- true
	}()
	return done
}

func GetBrokerDriver[T any]() T {

	if broker.config.Broker == "" {
		logger.New().Panic("the broker has not been initialized yet")
	}
	if broker.config.Broker == "redis" {
		return rdb.(T)
	}
	return errors.New("unknow driver").(T)
}

// consumer...
var (
	NilHandle = errors.New("beanq:handle is nil")
	NilCancel = errors.New("beanq:cancel is nil")
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
)
type (
	DefaultHandle struct {
		DoHandle func(ctx context.Context, message *Message) error
		DoCancel func(ctx context.Context, message *Message) error
		DoError  func(ctx context.Context, err error)
	}
	WorkflowHandler func(ctx context.Context, wf *Workflow) error
)

func (c WorkflowHandler) Handle(ctx context.Context, message *Message) error {
	workflow := NewWorkflow(ctx, message)
	return c(ctx, workflow)
}

func (c DefaultHandle) Handle(ctx context.Context, message *Message) error {
	if c.DoHandle != nil {
		return c.DoHandle(ctx, message)
	}
	return NilHandle
}

func (c DefaultHandle) Cancel(ctx context.Context, message *Message) error {
	if c.DoCancel != nil {
		return c.DoCancel(ctx, message)
	}
	return NilCancel
}

func (c DefaultHandle) Error(ctx context.Context, err error) {
	if c.DoError != nil {
		c.DoError(ctx, err)
	}
}
