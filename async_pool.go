package beanq

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	"github.com/retail-ai-inc/beanq/helper/logger"
)

type AsyncPool struct {
	pool *ants.Pool
}

func NewAsyncPool(poolSize int) *AsyncPool {
	pool, err := ants.NewPool(
		poolSize,
		ants.WithPreAlloc(true),
		ants.WithNonblocking(true),
		ants.WithPanicHandler(func(i interface{}) {
			logger.New().Error(i)

			localHub := sentry.CurrentHub().Clone()
			localHub.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetTag("goroutine", "true")
			})
			localHub.Recover(i)
		}))
	if err != nil {
		logger.New().With("", err).Panic("goroutine pool error")
	}

	return &AsyncPool{pool: pool}
}

func (a *AsyncPool) Execute(ctx context.Context, fn func(c context.Context) error, durations ...time.Duration) {
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

	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
		c = sentry.SetHubOnContext(c, hub)
	} else {
		c = sentry.SetHubOnContext(c, hub)
	}

	err := a.pool.Submit(func() {
		e := fn(c)
		if e != nil {
			captureException(ctx, e)
		}
	})

	if err != nil {
		captureException(ctx, errors.WithStack(err))
	}
}

func (a *AsyncPool) Release() {
	a.pool.Release()
}

func captureException(ctx context.Context, err error) {
	if err == nil {
		return
	}

	logger.New().Error(err)

	if sentry.CurrentHub().Client() == nil {
		return
	}

	if ctx != nil {
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			hub.CaptureException(err)
			return
		}
	}

	sentry.CurrentHub().Clone().CaptureException(err)
}

func recoverPanic(c context.Context) {
	if err := recover(); err != nil {
		logger.New().Error(err)

		if sentry.CurrentHub().Client() == nil {
			return
		}

		// Create a new Hub by cloning the existing one.
		var localHub *sentry.Hub

		if c != nil {
			localHub = sentry.GetHubFromContext(c)
		}

		if localHub == nil {
			localHub = sentry.CurrentHub().Clone()
		}

		localHub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("goroutine", "true")
		})

		localHub.Recover(err)
	}
}
