package public

import (
	"context"

	"github.com/retail-ai-inc/beanq/v4/helper/logger"
)

// Stream is the internal queue message envelope used by Redis stream workers.
type Stream struct {
	Data    map[string]any
	Id      string
	Channel string
	Stream  string
}

type CallbackWithRetry interface {
	Handle(ctx context.Context, data map[string]any, retry ...int) (int, error)
	Error(ctx context.Context, err error)
}

type callbackWithRetry struct {
	handle  func(ctx context.Context, data map[string]any, retry ...int) (int, error)
	onError func(ctx context.Context, err error)
}

func NewCallbackWithRetry(
	handle func(ctx context.Context, data map[string]any, retry ...int) (int, error),
	onError func(ctx context.Context, err error),
) CallbackWithRetry {
	return callbackWithRetry{handle: handle, onError: onError}
}

func (c callbackWithRetry) Handle(ctx context.Context, data map[string]any, retry ...int) (int, error) {
	if c.handle == nil {
		return 0, nil
	}
	return c.handle(ctx, data, retry...)
}

func (c callbackWithRetry) Error(ctx context.Context, err error) {
	if err == nil {
		return
	}
	if c.onError != nil {
		c.onError(ctx, err)
		return
	}
	logger.New().Error(err)
}
