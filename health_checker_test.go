package beanq

import (
	"context"
	"testing"

	opt "beanq/internal/options"
	"github.com/go-redis/redis/v8"
)

var (
	group           = "g2"
	consumer        = "cs1"
	optionParameter opt.Options
)

func init() {
	optionParameter = opt.Options{
		RedisOptions: &redis.Options{
			Addr:     "localhost:6381",
			Username: "",
			Password: "secret",
			DB:       0,
		},
	}
}

func TestStart(t *testing.T) {
	ctx := context.Background()
	check := newHealthCheck(redis.NewClient(optionParameter.RedisOptions))
	err := check.start(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestTest(t *testing.T) {
	check := newHealthCheck(redis.NewClient(optionParameter.RedisOptions))
	info, err := check.info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("info data:%+v \n", info)
}
