package bredis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
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
		tlsConfig, err = loadTLSConfigFromCA(h.caFile, h.verifyCertificate)
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

func loadTLSConfigFromCA(caFile string, verifyCertificate bool) (*tls.Config, error) {

	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA file %q: %w", caFile, err)
	}

	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caPEM); !ok {
		return nil, fmt.Errorf("CA file %q does not contain a valid PEM certificate", caFile)
	}

	return &tls.Config{
		RootCAs:            pool,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: verifyCertificate, // Do not set InsecureSkipVerify in production.
	}, nil
}

func WatchCAFile(ctx context.Context, caFile string, reload func(context.Context) error) error {

	absCAFile, err := filepath.Abs(caFile)
	if err != nil {
		return err
	}

	watchDir := filepath.Dir(absCAFile)
	watchBase := filepath.Base(absCAFile)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := watcher.Add(watchDir); err != nil {
		_ = watcher.Close()
		return err
	}

	go func() {
		defer watcher.Close()

		var (
			timer   *time.Timer
			timerMu sync.Mutex
		)

		triggerReload := func(reason string) {
			timerMu.Lock()
			defer timerMu.Unlock()

			if timer != nil {
				timer.Stop()
			}

			timer = time.AfterFunc(500*time.Millisecond, func() {
				reloadCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()

				logger.New().Debug("CA file changed, reloading redis client", "reason", reason)
				if err := reload(reloadCtx); err != nil {
					logger.New().Error("reload failed, keep using old redis client", "error", err)
				}
			})
		}

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if !isTargetCAEvent(event, watchDir, watchBase) {
					continue
				}

				if event.Has(fsnotify.Write) ||
					event.Has(fsnotify.Create) ||
					event.Has(fsnotify.Rename) ||
					event.Has(fsnotify.Remove) ||
					event.Has(fsnotify.Chmod) {
					triggerReload(event.String())
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.New().Debug("CA watcher error", "error", err)
			}
		}
	}()

	logger.New().Debug("watching CA file", "file", absCAFile)
	return nil
}

func isTargetCAEvent(event fsnotify.Event, watchDir, watchBase string) bool {

	if filepath.Base(event.Name) == watchBase {
		return true
	}

	// Kubernetes Secret/ConfigMap volumes update symlinks like "..data".
	name := filepath.Base(event.Name)
	if name == "..data" || name == "..data_tmp" {
		return true
	}

	// Some deploy tools replace the whole directory entry with a temp file.
	return filepath.Dir(event.Name) == watchDir && filepath.Base(event.Name) == watchBase
}
