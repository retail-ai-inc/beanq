package bredis

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/btls"
)

type RedisHolder struct {
	redis.UniversalClient
	mu                sync.RWMutex
	isCluster         bool
	host              string
	port              string
	username          string
	password          string
	db                int
	sslOn             bool
	caFile            string
	verifyCertificate bool
	maxRetries        int
	dialTimeout       time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	poolTimeout       time.Duration
	poolSize          int
	minIdleConns      int
}

func NewRedisHolder(ctx context.Context, isCluster bool, host, port, username, password string, db int,
	maxRetries int, dialTimeout,
	readTimeout, writeTimeout, poolTimeout time.Duration, poolSize,
	minIdleConns int,
	sslOn bool, caFile string, verifyCertificate bool) (*RedisHolder, error) {

	h := &RedisHolder{
		isCluster:         isCluster,
		host:              host,
		port:              port,
		username:          username,
		password:          password,
		db:                db,
		maxRetries:        maxRetries,
		dialTimeout:       dialTimeout,
		readTimeout:       readTimeout,
		writeTimeout:      writeTimeout,
		poolTimeout:       poolTimeout,
		poolSize:          poolSize,
		minIdleConns:      minIdleConns,
		sslOn:             sslOn,
		verifyCertificate: verifyCertificate,
		caFile:            caFile,
	}

	client, err := h.newClient(ctx)
	if err != nil {
		return nil, err
	}

	h.UniversalClient = client
	return h, nil
}

func (h *RedisHolder) Client() redis.UniversalClient {

	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.UniversalClient
}

func (h *RedisHolder) Reload(ctx context.Context) error {

	newClient, err := h.newClient(ctx)
	if err != nil {
		return err
	}
	h.mu.Lock()
	oldClient := h.UniversalClient
	h.UniversalClient = newClient
	h.mu.Unlock()
	if oldClient != nil {
		_ = oldClient.Close()
	}

	return nil
}

func (h *RedisHolder) Close() error {

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.UniversalClient == nil {
		return nil
	}

	err := h.UniversalClient.Close()
	h.UniversalClient = nil
	return err
}

func (h *RedisHolder) newClient(ctx context.Context) (redis.UniversalClient, error) {

	var tlsConfig *tls.Config
	if h.sslOn {
		var err error
		tlsConfig, err = btls.LoadTLSConfigFromCA(h.caFile, h.verifyCertificate)
		if err != nil {
			return nil, err
		}
	}

	hosts := strings.Split(h.host, ",")
	for i, host := range hosts {
		hs := strings.Split(host, ":")
		if len(hs) == 1 {
			hosts[i] = strings.Join([]string{host, h.port}, ":")
		}
	}

	var client redis.UniversalClient
	if h.isCluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:          hosts,
			Username:       h.username,
			Password:       h.password,
			MaxRetries:     h.maxRetries,
			DialTimeout:    h.dialTimeout,
			ReadTimeout:    h.readTimeout,
			WriteTimeout:   h.writeTimeout,
			PoolSize:       h.poolSize,
			MinIdleConns:   h.minIdleConns,
			PoolTimeout:    h.poolTimeout,
			RouteByLatency: true,

			TLSConfig: tlsConfig,
		})
	} else {
		client = redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:          hosts,
			Username:       h.username,
			Password:       h.password,
			DB:             h.db,
			MaxRetries:     h.maxRetries,
			DialTimeout:    h.dialTimeout,
			ReadTimeout:    h.readTimeout,
			WriteTimeout:   h.writeTimeout,
			PoolSize:       h.poolSize,
			MinIdleConns:   h.minIdleConns,
			PoolTimeout:    h.poolTimeout,
			RouteByLatency: true,

			TLSConfig: tlsConfig,
		})
	}

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return client, nil
}
