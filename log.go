package beanq

import (
	"context"

	"github.com/go-redis/redis/v8"
	"golang.org/x/sync/errgroup"
)

type ILog interface {
	// Save log
	Save(ctx context.Context, log ILog) error
	// Obsolete ,if log has expired ,then delete it
	Obsolete(ctx context.Context, log ILog) error
}

type DefaultLog struct {
	egroup *errgroup.Group
}

func NewDefaultLog() *DefaultLog {
	egroup := new(errgroup.Group)
	egroup.SetLimit(2)
	return &DefaultLog{
		egroup: egroup,
	}
}

func (t *DefaultLog) Save(ctx context.Context, log ILog) error {
	t.egroup.TryGo(func() error {
		if log != nil {
			if err := log.Save(ctx, nil); err != nil {
				return err
			}
		}
		return nil
	})
	t.egroup.TryGo(func() error {
		// default save log logic
		// ...
		return nil
	})
	return t.egroup.Wait()
}

func (t *DefaultLog) Obsolete(ctx context.Context, log ILog) error {
	t.egroup.TryGo(func() error {
		if log != nil {
			if err := log.Obsolete(ctx, nil); err != nil {
				return err
			}
		}
		return nil
	})
	t.egroup.TryGo(func() error {
		// default obsolete logic
		// ...
		return nil
	})
	return t.egroup.Wait()
}

type RedisLog struct {
	client redis.UniversalClient
}

func (t *RedisLog) Save(ctx context.Context, log ILog) error {
	return nil
}

func (t *RedisLog) Obsolete(ctx context.Context, log ILog) error {
	return nil
}
