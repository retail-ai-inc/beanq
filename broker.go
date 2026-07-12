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

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/bredis"

	"github.com/retail-ai-inc/beanq/v4/helper/logger"
)

var (
	brokerOnce sync.Once
	broker     Broker
)

type Handler struct {
	brokerImpl queueGateway
	do         public.CallbackWithRetry
	channel    string
	topic      string
	moodType   btype.MoodType
	broker     string
	prefix     string
	retryCond  map[string]struct{}
}

func (h *Handler) Invoke(ctx context.Context, broker queueGateway) {
	broker.Dequeue(ctx, h.channel, h.topic, public.NewCallbackWithRetry(func(ctx context.Context, data map[string]any, retry ...int) (int, error) {
		if len(retry) == 0 {
			return h.do.Handle(ctx, data)
		}

		return tool.RetryInfo(ctx, func() error {
			_, err := h.do.Handle(ctx, data)
			return err
		}, retry[0], func(err error) bool {
			key := fmt.Sprintf("%T,%v", err, err.Error())
			if _, ok := h.retryCond[key]; ok {
				return true
			}
			return false
		})
	}, h.do.Error))
}

type Broker struct {
	queue         queueGateway
	locker        locker
	status        statusReader
	client        any
	strategy      brokerStrategy
	config        *BeanqConfig
	tool          *bredis.UITool
	handlers      []*Handler
	captureConfig *capture.Config
}

func NewBroker(config *BeanqConfig) *Broker {
	brokerOnce.Do(func() {
		components, err := defaultBrokerBuilder(config)
		logBrokerBuildFailure(err, config.Broker)
		broker.queue = components.queue
		broker.locker = components.locker
		broker.status = components.status
		broker.client = components.client
		broker.strategy = components.strategy
		broker.tool = components.tool
		broker.captureConfig = components.captureConfig
	})
	broker.config = config
	return &broker
}

type captureConfigReader interface {
	ConfigInfo(ctx context.Context) (*capture.Config, error)
}

func getConfig(client captureConfigReader) *capture.Config {

	ctx, cancel := context.WithTimeout(context.Background(), configLookupTimeout())
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

func (t *Broker) ForceUnlock(ctx context.Context, channel, topic, orderKey string) error {
	if t.locker == nil {
		return ErrUnsupportedBroker.WithMessage("force unlock")
	}
	return t.locker.ForceUnlock(ctx, channel, topic, orderKey)
}

func (t *Broker) Enqueue(ctx context.Context, data map[string]any) error {
	if t.queue == nil {
		return ErrUnsupportedBroker.WithMessage("enqueue")
	}
	return t.queue.Enqueue(ctx, data)
}

func (t *Broker) Dequeue(ctx context.Context, channel, topic string, do public.CallbackWithRetry) {
}

func (t *Broker) Status(ctx context.Context, channel, topic, id string, isOrder bool) (map[string]string, error) {
	if t.status == nil {
		return nil, ErrUnsupportedBroker.WithMessage("status")
	}
	return t.status.Status(ctx, channel, topic, id, isOrder)
}

func (t *Broker) AddConsumer(moodType btype.MoodType, channel, topic string, subscribe IConsumeHandle) error {

	handler := Handler{
		broker:   t.config.Broker,
		prefix:   t.config.Redis.Prefix,
		channel:  channel,
		topic:    topic,
		moodType: moodType,
		do: public.NewCallbackWithRetry(func(ctx context.Context, message map[string]any, retry ...int) (int, error) {
			var gerr error
			msg := messageToStruct(message)
			if err := subscribe.Handle(ctx, msg); err != nil {
				gerr = errors.Join(gerr, err)
				if h, ok := subscribe.(IConsumeCancel); ok {
					gerr = errors.Join(gerr, h.Cancel(ctx, msg))
				}
			}
			return 0, gerr
		}, func(ctx context.Context, err error) {
			if h, ok := subscribe.(IConsumeError); ok {
				h.Error(ctx, err)
			}
		}),
	}
	handler.brokerImpl = t.strategy(moodType, t.captureConfig)
	t.handlers = append(t.handlers, &handler)

	return nil
}

func (t *Broker) Migrate(ctx context.Context, data []map[string]any) error {
	var migrate MigrationRunner

	if t.config.Broker == "redis" {
		migrate = newRedisMigrateLog(ctx, t.config, t.client.(redis.UniversalClient))
	}
	if migrate == nil {
		return ErrUnsupportedBroker.WithMessage(t.config.Broker)
	}

	return migrate.Migrate(ctx, data)
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

func (t *Broker) Mood(m btype.MoodType) queueGateway {
	return t.strategy(m, t.captureConfig)
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
	//nolint:staticcheck
	ErrNilHandle = errors.New("beanq:handle is nil")
	//nolint:staticcheck
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

// Migrate this will display nothing for the logs on ui-side.
func (discard) Migrate(ctx context.Context, data []map[string]any) error {
	return nil
}
