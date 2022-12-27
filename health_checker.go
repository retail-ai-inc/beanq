package beanq

import (
	"context"

	"beanq/helper/devicex"
	"beanq/helper/json"
	"github.com/go-redis/redis/v8"
)

type healthCheckI interface {
	start(ctx context.Context) error
}

type healthCheck struct {
	client *redis.Client
}

func newHealthCheck(client *redis.Client) *healthCheck {
	return &healthCheck{client: client}
}

func (t *healthCheck) start(ctx context.Context) (err error) {
	device := devicex.Device
	key := "health_checker"
	var str string

	if err = device.Info(); err != nil {
		return
	}
	if err = t.client.HDel(ctx, key, device.Net.Ip).Err(); err != nil {
		return
	}

	if str, err = json.Json.MarshalToString(device); err != nil {
		return
	}
	if err = t.client.HMSet(ctx, key, device.Net.Ip, str).Err(); err != nil {
		return
	}

	return nil
}
