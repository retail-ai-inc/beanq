package bredis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"strings"
	"sync"
	"time"
)

var (
	rdb     redis.UniversalClient
	rdbOnce sync.Once
)

func NewRdb(host, port string, password string,
	database, maxRetries int, dialTimeout,
	readTimeout, writeTimeout, poolTimeout time.Duration, poolSize, minIdleConns int) redis.UniversalClient {

	rdbOnce.Do(func() {
		ctx := context.Background()

		hosts := strings.Split(host, ",")
		for i, h := range hosts {
			hs := strings.Split(h, ":")
			if len(hs) == 1 {
				hosts[i] = strings.Join([]string{h, port}, ":")
			}
		}

		rdb = redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:        hosts,
			Password:     password,
			DB:           database,
			MaxRetries:   maxRetries,
			DialTimeout:  dialTimeout,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			PoolSize:     poolSize,
			MinIdleConns: minIdleConns,
			PoolTimeout:  poolTimeout,
			PoolFIFO:     false,
		})

		if err := rdb.Ping(ctx).Err(); err != nil {
			logger.New().Fatal(err.Error())
		}
	})

	return rdb
}