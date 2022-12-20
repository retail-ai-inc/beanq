package beanq

import (
	"context"
	"log"
	"time"

	"beanq/client"
	"beanq/driver"
	"beanq/server"
	"beanq/task"
)

type Beanq interface {
	Publish(ctx context.Context, task *task.Task, option ...client.Option) (*task.Result, error)
	DelayPublish(ctx context.Context, task *task.Task, delayTime time.Time, option ...client.Option) (*task.Result, error)
	Start(server *server.Server)
	StartUI() error
	Close() error
}

func NewBeanq(broker string, options task.Options) Beanq {

	if options.KeepJobInQueue == 0 {
		options.KeepJobInQueue = task.DefaultOptions.KeepJobInQueue
	}
	if options.KeepFailedJobsInHistory == 0 {
		options.KeepFailedJobsInHistory = task.DefaultOptions.KeepFailedJobsInHistory
	}
	if options.KeepSuccessJobsInHistory == 0 {
		options.KeepSuccessJobsInHistory = task.DefaultOptions.KeepSuccessJobsInHistory
	}
	if options.MinWorkers == 0 {
		options.MinWorkers = task.DefaultOptions.MinWorkers
	}
	if options.JobMaxRetry == 0 {
		options.JobMaxRetry = task.DefaultOptions.JobMaxRetry
	}
	if options.Prefix == "" {
		options.Prefix = task.DefaultOptions.Prefix
	}
	if broker == "redis" {
		if options.RedisOptions == nil {
			log.Fatalln("Missing Redis configuration")
		}
		return driver.NewRedis(options)
	}
	return nil
}
