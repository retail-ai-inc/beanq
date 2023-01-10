package beanq

import (
	"context"
	"testing"

	"beanq/internal/driver"
	opt "beanq/internal/options"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
)

var (
	group           = "g2"
	consumer        = "cs1"
	optionParameter opt.Options
)

func init() {
	optionParameter = opt.Options{
		RedisOptions: &redis.Options{
			Addr:      Config.Queue.Redis.Host + ":" + cast.ToString(Config.Queue.Redis.Port),
			Dialer:    nil,
			OnConnect: nil,
			Username:  "",
			Password:  Config.Queue.Redis.Password,
			DB:        Config.Queue.Redis.Database,
		},
		KeepJobInQueue:           Config.Queue.KeepJobsInQueue,
		KeepFailedJobsInHistory:  Config.Queue.KeepFailedJobsInHistory,
		KeepSuccessJobsInHistory: Config.Queue.KeepSuccessJobsInHistory,
		MinWorkers:               Config.Queue.MinWorkers,
		JobMaxRetry:              Config.Queue.JobMaxRetries,
		Prefix:                   Config.Queue.Redis.Prefix,
	}
}
func TestStart(t *testing.T) {
	ctx := context.Background()
	check := newHealthCheck(driver.NewRdb(optionParameter.RedisOptions))
	err := check.start(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}
}
