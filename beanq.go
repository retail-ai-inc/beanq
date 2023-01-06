package beanq

import (
	"context"
	"time"

	opt "beanq/internal/options"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
)

type BeanqPub interface {
	Publish(task *Task, option ...opt.OptionI) (*opt.Result, error)
	PublishContext(ctx context.Context, task *Task, option ...opt.OptionI) (*opt.Result, error)
	DelayPublish(task *Task, delayTime time.Time, option ...opt.OptionI) (*opt.Result, error)
	Close() error
}
type BeanqSub interface {
	Start(server *Server)
	StartContext(ctx context.Context, srv *Server)
	StartUI() error
}

type Broker interface {
	enqueue(ctx context.Context, stream string, task *Task, options opt.Option) (*opt.Result, error)
	close() error
	start(ctx context.Context, server *Server)
}

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

// TODO
func Consume(server *Server, opts *opt.Options) error {
	return nil
}
