package redisx

import (
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	redisOnce sync.Once
	client    *redis.Client
)

func Client(addr, password string, db int) *redis.Client {

	redisOnce.Do(func() {
		client = redis.NewClient(&redis.Options{
			Network:  "",
			Addr:     addr,
			Username: "",
			Password: password,
			DB:       db,
		})
	})

	return client
}
