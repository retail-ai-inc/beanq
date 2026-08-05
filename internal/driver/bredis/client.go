package bredis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/btls"
)

type RedisClientOptions struct {
	IsCluster          bool
	Host               string
	Port               string
	Username           string
	Password           string
	Database           int
	MaxRetries         int
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	PoolTimeout        time.Duration
	PoolSize           int
	MinIdleConnections int
	TLS                RedisTLSOptions
	WaitMode           string
	WaitReplicas       int
	WaitAOFLocal       int
}

type RedisTLSOptions struct {
	On                bool
	CAFile            string
	VerifyCertificate bool
	HotReload         bool
}

// NewRdb preserves the legacy positional-argument API.
// Deprecated: use NewRedisClient.
func NewRdb(isCluster bool, host, port string, username, password string,
	database, maxRetries int, dialTimeout,
	readTimeout, writeTimeout, poolTimeout time.Duration, poolSize,
	minIdleConns int,
	sslOn bool, caFile string, verifyCertificate bool, hotReload bool, waitReplicas int) (redis.UniversalClient, error) {

	return NewRedisClient(context.Background(), RedisClientOptions{
		IsCluster: isCluster, Host: host, Port: port, Username: username, Password: password,
		Database: database, MaxRetries: maxRetries, DialTimeout: dialTimeout, ReadTimeout: readTimeout,
		WriteTimeout: writeTimeout, PoolTimeout: poolTimeout, PoolSize: poolSize, MinIdleConnections: minIdleConns,
		TLS:      RedisTLSOptions{On: sslOn, CAFile: caFile, VerifyCertificate: verifyCertificate, HotReload: hotReload},
		WaitMode: WaitModeReplication, WaitReplicas: waitReplicas,
	})
}

func NewRedisClient(ctx context.Context, options RedisClientOptions) (*RedisHolder, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	initCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	holder, err := NewRedisHolderWithOptions(initCtx, options)
	if err != nil {
		return nil, err
	}
	if err := ValidateWaitTopology(initCtx, holder.UniversalClient, options.WaitMode, options.WaitReplicas, options.WaitAOFLocal); err != nil {
		_ = holder.Close()
		return nil, err
	}
	if options.TLS.HotReload && options.TLS.On && options.TLS.CAFile != "" {
		watchCtx, stopWatching := context.WithCancel(ctx)
		if err := btls.WatchCAFile(watchCtx, `redis`, options.TLS.CAFile, holder.Reload); err != nil {
			stopWatching()
			_ = holder.Close()
			return nil, err
		}
		holder.watcherCancel = stopWatching
	}
	return holder, nil
}
