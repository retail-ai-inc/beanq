package beanq

import (
	"log"

	"beanq/client"
	"beanq/driver"
	"beanq/task"
)

func NewBeanq(broker string, options task.Options) driver.Beanq {

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
		return client.NewRedis(options)
	}
	return nil
}
