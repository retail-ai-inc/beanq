package beanq

import (
	"context"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
)

type AsyncExecutor interface {
	Execute(ctx context.Context, fn func(c context.Context) error, durations ...time.Duration)
	Release()
}

type AsyncPoolOption func(*asyncPool)

type asyncPool struct {
	pool             *ants.Pool
	captureException func(ctx context.Context, err any)
}

func newAsyncPool(poolSize int, options ...AsyncPoolOption) *asyncPool {
	var (
		pool *ants.Pool
		err  error
	)

	if poolSize < 0 {
		pool, err = ants.NewPool(-1)
	} else {
		pool, err = ants.NewPool(
			poolSize,
			ants.WithPreAlloc(true))
	}

	if err != nil {
		logger.New().With("", err).Panic("goroutine pool error")
	}

	ap := &asyncPool{
		pool:             pool,
		captureException: defaultCaptureException,
	}
	for _, option := range options {
		option(ap)
	}
	return ap
}

func WithAsyncCaptureException(handler func(ctx context.Context, err any)) AsyncPoolOption {
	return func(pool *asyncPool) {
		if handler != nil {
			pool.captureException = handler
		}
	}
}

func (a *asyncPool) Execute(ctx context.Context, fn func(c context.Context) error, durations ...time.Duration) {
	var (
		cancel context.CancelFunc
	)
	if len(durations) > 0 {
		ctx, cancel = context.WithTimeout(ctx, durations[0])
		defer cancel()
	}

	err := a.pool.Submit(func() {
		defer func() {
			if err := recover(); err != nil {
				a.captureException(ctx, err)
			}
		}()

		e := fn(ctx)
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
