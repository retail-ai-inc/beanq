package beanq

import (
	"context"
	"errors"
	"sync"

	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	public "github.com/retail-ai-inc/beanq/v3/internal"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
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
	do              func(ctx context.Context, data map[string]any) error
	channel         string
	topic           string
	moodType        btype.MoodType
	retryConditions []func(error) bool
}

func (h *Handler) Invoke(ctx context.Context, broker public.IBroker) {
	broker.Dequeue(ctx, h.channel, h.topic, func(ctx context.Context, data map[string]any, retry ...int) (int, error) {
		if len(retry) == 0 {
			return 0, h.do(ctx, data)
		}

		return tool.RetryInfo(ctx, func() error {
			return h.do(ctx, data)
		}, retry[0], h.retryConditions...)
	})
}

type Broker struct {
	status   public.IStatus
	log      public.IProcessLog
	client   any
	fac      public.IBrokerFactory
	config   *BeanqConfig
	tool     *bredis.UITool
	handlers []*Handler
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
		default:
			logger.New().Panic("not support broker type:", config.Broker)
		}
	})
	broker.config = config
	return &broker
}

func (t *Broker) Enqueue(ctx context.Context, data map[string]any) error {
	moodType := btype.NORMAL

	if v, ok := data["moodType"]; ok {
		moodType = btype.MoodType(cast.ToString(v))
	}

	bk := t.fac.Mood(moodType)
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

func (t *Broker) Status(ctx context.Context, channel, topic, id string) (map[string]string, error) {
	data, err := t.status.Status(ctx, channel, topic, id)
	if err != nil {
		// todo
		return nil, err
	}
	return data, nil
}

func (t *Broker) Migrate(ctx context.Context) error {
	migrate := MigrateLogDiscard

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

	return t.fac.Migrate(ctx, migrate)
}

func (t *Broker) Mood(m btype.MoodType) public.IBroker {
	return t.fac.Mood(m)
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

var MigrateLogDiscard public.IMigrateLog = discard{}

type discard struct{}

// Migrate this will display nothing for the logs on ui-side.
func (discard) Migrate(ctx context.Context, data []map[string]any) error {
	return nil
}
