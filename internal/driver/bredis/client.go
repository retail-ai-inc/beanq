package bredis

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/btls"
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
	sslOn bool, caFile string, verifyCertificate bool, hotReload bool, waitReplicas int) (redis.UniversalClient, error) {

	redisOnce.Do(func() {
		ctx := context.Background()
		//Preventing latency caused by Redis Cluster network topology
		initCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		redisHolder, redisErr = NewRedisHolder(
			initCtx, isCluster, host, port, username, password, database,
			maxRetries, dialTimeout, readTimeout, writeTimeout, poolTimeout, poolSize, minIdleConns,
			sslOn, caFile, verifyCertificate)
		if redisErr != nil {
			return
		}
		if redisErr = ValidateReplicationTopology(initCtx, redisHolder.UniversalClient, waitReplicas); redisErr != nil {
			_ = redisHolder.Close()
			redisHolder = nil
			return
		}
		if hotReload && sslOn && caFile != "" {
			redisErr = btls.WatchCAFile(ctx, "redis", caFile, redisHolder.Reload)
			if redisErr != nil {
				_ = redisHolder.Close()
				redisHolder = nil
			}
		}
	})
	return redisHolder, redisErr
}
