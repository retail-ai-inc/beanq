package bredis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/btls"
)

type RedisHolder struct {
	redis.UniversalClient
	caPool            atomic.Pointer[x509.CertPool]
	closeOnce         sync.Once
	closeErr          error
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

	if h.sslOn {
		if err := h.Reload(ctx); err != nil {
			return nil, err
		}
	}
	client, err := h.newClient(ctx)
	if err != nil {
		return nil, err
	}

	h.UniversalClient = client
	return h, nil
}

// Client exposes the current underlying client to packages that need to
// distinguish standalone and Cluster Redis clients.
func (h *RedisHolder) Client() redis.UniversalClient {
	return h.UniversalClient
}

func (h *RedisHolder) PipelinedForKey(ctx context.Context, key string, targetAddress string, asking bool, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	if h.UniversalClient == nil {
		return nil, fmt.Errorf("redis client is not initialized")
	}
	return pipelinedForClient(ctx, h.UniversalClient, key, targetAddress, asking, fn)
}

func (h *RedisHolder) Reload(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("reload Redis CA: %w", err)
	}
	tlsConfig, err := btls.LoadTLSConfigFromCA(h.caFile, h.verifyCertificate)
	if err != nil {
		return fmt.Errorf("reload Redis CA: %w", err)
	}
	h.caPool.Store(tlsConfig.RootCAs)
	return nil
}

func (h *RedisHolder) Close() error {
	h.closeOnce.Do(func() {
		if h.UniversalClient != nil {
			h.closeErr = h.UniversalClient.Close()
		}
	})
	return h.closeErr
}

func (h *RedisHolder) newClient(ctx context.Context) (redis.UniversalClient, error) {

	var tlsConfig *tls.Config
	if h.sslOn {
		tlsConfig = h.tlsConfig()
	}

	hosts, err := redisAddresses(h.host, h.port)
	if err != nil {
		return nil, err
	}

	options := &redis.UniversalOptions{
		Addrs:        hosts,
		Username:     h.username,
		Password:     h.password,
		DB:           h.db,
		MaxRetries:   h.maxRetries,
		DialTimeout:  h.dialTimeout,
		ReadTimeout:  h.readTimeout,
		WriteTimeout: h.writeTimeout,
		PoolSize:     h.poolSize,
		MinIdleConns: h.minIdleConns,
		PoolTimeout:  h.poolTimeout,
		TLSConfig:    tlsConfig,
	}

	var client redis.UniversalClient
	if h.isCluster {
		client = redis.NewClusterClient(options.Cluster())
	} else {
		client = redis.NewUniversalClient(options)
	}

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return client, nil
}

func redisAddresses(hosts, defaultPort string) ([]string, error) {
	defaultPort = strings.TrimPrefix(strings.TrimSpace(defaultPort), ":")
	addresses := strings.Split(hosts, ",")
	result := make([]string, 0, len(addresses))

	for _, address := range addresses {
		address = strings.TrimSpace(address)
		if address == "" {
			return nil, fmt.Errorf("redis host contains an empty address")
		}

		if host, port, err := net.SplitHostPort(address); err == nil {
			result = append(result, net.JoinHostPort(host, port))
			continue
		}

		host := strings.TrimSuffix(strings.TrimPrefix(address, "["), "]")
		if strings.Contains(address, ":") && net.ParseIP(host) == nil {
			return nil, fmt.Errorf("invalid redis address %q", address)
		}
		if defaultPort == "" {
			return nil, fmt.Errorf("redis address %q does not include a port", address)
		}
		result = append(result, net.JoinHostPort(host, defaultPort))
	}

	return result, nil
}

func (h *RedisHolder) tlsConfig() *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, // Verification uses the atomically reloadable CA pool below.
		VerifyConnection: func(state tls.ConnectionState) error {
			if !h.verifyCertificate {
				return nil
			}
			if len(state.PeerCertificates) == 0 {
				return fmt.Errorf("verify Redis TLS connection: peer sent no certificates")
			}

			intermediates := x509.NewCertPool()
			for _, certificate := range state.PeerCertificates[1:] {
				intermediates.AddCert(certificate)
			}
			_, err := state.PeerCertificates[0].Verify(x509.VerifyOptions{
				DNSName:       state.ServerName,
				Roots:         h.caPool.Load(),
				Intermediates: intermediates,
				KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			})
			if err != nil {
				return fmt.Errorf("verify Redis TLS connection: %w", err)
			}
			return nil
		},
	}
}
