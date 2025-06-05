package beanq

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	bmongo2 "github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/internal"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/retail-ai-inc/beanq/v3/internal/capture"
	"github.com/retail-ai-inc/beanq/v3/internal/driver/bmongo"
	"github.com/retail-ai-inc/beanq/v3/internal/driver/bredis"
	"github.com/spf13/cast"

	"github.com/retail-ai-inc/beanq/v3/helper/logger"
)

var (
	brokerOnce sync.Once
	broker     Broker
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
	status        public.IStatus
	log           public.IProcessLog
	client        any
	fac           public.IBrokerFactory
	config        *BeanqConfig
	tool          *bredis.UITool
	handlers      []*Handler
	captureConfig *capture.Config
}

func NewBroker(config *BeanqConfig) *Broker {

	brokerOnce.Do(func() {
		switch config.Broker {
		case "redis":
			cfg := config.Redis
			client := bredis.NewRdb(cfg.Host, cfg.Port,
				cfg.Password, cfg.Database,
				cfg.MaxRetries, cfg.DialTimeout, cfg.ReadTimeout, cfg.WriteTimeout, cfg.PoolTimeout, cfg.PoolSize, cfg.MinIdleConnections)

			broker.status = bredis.NewStatus(client, cfg.Prefix)
			broker.log = bredis.NewProcessLog(client, cfg.Prefix)
			broker.client = client
			broker.fac = bredis.NewBroker(client, cfg.Prefix, cfg.MaxLen, cfg.MaxLen, config.DeadLetterIdleTime)
			broker.tool = bredis.NewUITool(client, cfg.Prefix)
			// capture errors and send them to email or Slack

			if config.History.On {
				mcfg := config.Mongo
				nmgo := bmongo2.NewMongo(mcfg.Host,
					mcfg.Port, mcfg.UserName,
					mcfg.Password,
					mcfg.Database,
					mcfg.Collections,
					mcfg.ConnectTimeOut,
					mcfg.MaxConnectionPoolSize,
					mcfg.MaxConnectionLifeTime)

				broker.captureConfig = getConfig(nmgo)
			}

		default:
			logger.New().Panic("not support broker type:", config.Broker)
		}

	})
	broker.config = config
	return &broker
}

func getConfig(client *bmongo2.BMongo) *capture.Config {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if client == nil {
		return nil
	}
	cfg, err := client.ConfigInfo(ctx)
	if err != nil {
		return nil
	}
	return cfg
}

func (t *Broker) Enqueue(ctx context.Context, data map[string]any) error {

	moodType := btype.NORMAL

	if v, ok := data["moodType"]; ok {
		moodType = btype.MoodType(cast.ToString(v))
	}

	bk := t.fac.Mood(moodType, t.captureConfig)
	if bk == nil {
		return bstatus.BrokerDriverError
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

func (t *Broker) Status(ctx context.Context, channel, topic, id string, isOrder bool) (map[string]string, error) {

	data, err := t.status.Status(ctx, channel, topic, id, isOrder)
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

			var gerr error
			msg := messageToStruct(message)
			if err := subscribe.Handle(ctx, msg); err != nil {
				gerr = errors.Join(gerr, err)
				if h, ok := subscribe.(IConsumeCancel); ok {
					gerr = errors.Join(gerr, h.Cancel(ctx, msg))
				}
			}
			return gerr
		},
	}
	handler.brokerImpl = t.fac.Mood(moodType, t.captureConfig)
	t.handlers = append(t.handlers, &handler)

	return nil
}

func (t *Broker) Migrate(ctx context.Context, data []map[string]any) error {

	var migrate public.IMigrateLog

	if t.config.Broker == "redis" {
		if t.config.History.On {
			mongo := t.config.Mongo
			migrate = bmongo.NewMongoLog(ctx,
				mongo.Host,
				mongo.Port,
				mongo.ConnectTimeOut,
				mongo.MaxConnectionLifeTime,
				mongo.MaxConnectionPoolSize,
				mongo.Database,
				mongo.Collections["event"],
				mongo.UserName,
				mongo.Password)
		}
		migrate = bredis.NewLog(t.client.(redis.UniversalClient), t.config.Redis.Prefix, migrate)
	}

	return migrate.Migrate(ctx, nil)

}

func (t *Broker) Start(ctx context.Context) {

	ctx, cancel := context.WithCancel(ctx)

	for key, handler := range t.handlers {
		hdl := *handler
		go func(hdl2 Handler) {
			hdl2.brokerImpl.Dequeue(ctx, hdl2.channel, hdl2.topic, hdl2.do)
		}(hdl)
		t.handlers[key] = nil
	}
	//move logs from redis to mongo
	go func() {
		_ = t.Migrate(ctx, nil)
	}()
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ticker.Reset(10 * time.Second)
				if err := t.tool.HostName(ctx); err != nil {
					fmt.Printf("hostname err:%+v \n", err)
				}
			}
		}
	}()

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
		return broker.client.(T)
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
