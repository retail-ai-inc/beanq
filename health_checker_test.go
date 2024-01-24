package beanq

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
)

var (
	optionParameter Options
)

func init() {
	optionParameter = Options{
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
func TestHealthData(t *testing.T) {
	check := newHealthCheck(redis.NewClient(optionParameter.RedisOptions))
	info, err := check.info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	data, _ := info.toHealthData()
	t.Fatalf("info data:%+v \n", data)
}
