package driver

import (
	"context"
	"time"

	"beanq/server"
	"beanq/task"
)

type Beanq interface {
	Publish(ctx context.Context, task *task.Task, option ...Option) (*task.Result, error)
	DelayPublish(ctx context.Context, task *task.Task, delayTime time.Time, option ...Option) (*task.Result, error)
	Start(server *server.Server)
	StartUI() error
	Close() error
}
