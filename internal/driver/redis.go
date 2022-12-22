package driver

import (
	"sync"

	"github.com/go-redis/redis/v8"
)

var (
	once        sync.Once
	redisClient *redis.Client
)

func NewRdb(options *redis.Options) *redis.Client {
	once.Do(func() {
		redisClient = redis.NewClient(options)
	})
	return redisClient
}
