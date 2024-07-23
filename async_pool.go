package beanq

import (
	"context"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	"github.com/retail-ai-inc/beanq/helper/logger"
)

type asyncPool struct {
	pool             *ants.Pool
	captureException func(ctx context.Context, err any)
}

func newAsyncPool(poolSize int) *asyncPool {
	pool, err := ants.NewPool(
		poolSize,
		ants.WithPreAlloc(true))
	if err != nil {
		logger.New().With("", err).Panic("goroutine pool error")
	}

	return &asyncPool{
		pool:             pool,
		captureException: defaultCaptureException,
	}
}

func (a *asyncPool) Execute(ctx context.Context, fn func(c context.Context) error, durations ...time.Duration) {
	var (
		c      context.Context
		cancel context.CancelFunc
	)
	if len(durations) > 0 {
		c, cancel = context.WithTimeout(context.TODO(), durations[0])
		defer cancel()
	} else {
		c = context.TODO()
	}

	err := a.pool.Submit(func() {
		defer func() {
			if err := recover(); err != nil {
				a.captureException(ctx, err)
			}
		}()

		e := fn(c)
		if e != nil {
			a.captureException(ctx, e)
		}
	})

	if err != nil {
		a.captureException(ctx, errors.WithStack(err))
	}
}

func (a *asyncPool) Release() {
	a.pool.Release()
}

var defaultCaptureException = func(ctx context.Context, err any) {
	if err == nil {
		return
	}

	logger.New().Error(err)
}
