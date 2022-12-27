package beanq

import (
	"context"
	"time"

	opt "beanq/internal/options"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
)

type Beanq interface {
	Publish(task *Task, option ...opt.Option) (*opt.Result, error)
	DelayPublish(task *Task, delayTime time.Time, option ...opt.Option) (*opt.Result, error)
	Start(server *Server)
	StartUI() error
	Close() error
}

type Broker interface {
	Enqueue(ctx context.Context, values map[string]any, options opt.Option) (*opt.Result, error)
	Close() error
	Start(ctx context.Context, server *Server)
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
