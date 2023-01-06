package beanq

import (
	"context"
	"time"

	"beanq/helper/logger"
	opt "beanq/internal/options"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
)

type BeanqPub interface {
	Publish(task *Task, option ...opt.OptionI) (*opt.Result, error)
	PublishWithContext(ctx context.Context, task *Task, option ...opt.OptionI) (*opt.Result, error)
	DelayPublish(task *Task, delayTime time.Time, option ...opt.OptionI) (*opt.Result, error)
}

// This is a global variable to hold the debug logger so that we can log data from anywhere.
var Logger logger.Logger

// Hold the useful configuration settings of beanq so that we can use it quickly from anywhere.
var Config BeanqConfig

type BeanqSub interface {
	StartConsumer(server *Server)
	StartConsumerWithContext(ctx context.Context, srv *Server)
	StartUI() error
}

type Broker interface {
	enqueue(ctx context.Context, stream string, task *Task, options opt.Option) (*opt.Result, error)
	close() error
	start(ctx context.Context, server *Server)
}

// easy publish
// only input Task and set options
func Publish(task *Task, opts ...opt.OptionI) error {
	redisOpts := &redis.Options{
		Addr:     Env.Queue.Redis.Host + ":" + cast.ToString(Env.Queue.Redis.Port),
		Password: Env.Queue.Redis.Password,
		DB:       Env.Queue.Redis.Db,
	}

	pub := NewClient(NewRedisBroker(redisOpts))
	_, err := pub.Publish(task, opts...)
	if err != nil {
		return err
	}

	defer pub.Close()
	return nil
}

// easy consume
// heavily  to implement
func Consume(server *Server, opts *opt.Options) error {
	return nil
}
