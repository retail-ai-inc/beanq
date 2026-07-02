package bredis

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
)

var (
	redisOnce   sync.Once
	redisHolder *RedisHolder
	redisErr    error
)

func NewRdb(isCluster bool, host, port string, username, password string,
	database, maxRetries int, dialTimeout,
	readTimeout, writeTimeout, poolTimeout time.Duration, poolSize,
	minIdleConns int,
	sslOn bool, caFile string, verifyCertificate bool, hotReload bool) (redis.UniversalClient, error) {

	redisOnce.Do(func() {
		ctx := context.Background()
		initCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		redisHolder, redisErr = NewRedisHolder(
			initCtx, isCluster, host, port, username, password, database,
			maxRetries, dialTimeout, readTimeout, writeTimeout, poolTimeout, poolSize, minIdleConns,
			sslOn, caFile, verifyCertificate)
		if redisErr != nil {
			logger.New().Fatal(redisErr.Error())
		}
		if hotReload && sslOn && caFile != "" {
			redisErr = WatchCAFile(ctx, caFile, redisHolder.Reload)
			if redisErr != nil {
				_ = redisHolder.Close()
				redisHolder = nil
			}
		}
	})
	return redisHolder, redisErr
}
