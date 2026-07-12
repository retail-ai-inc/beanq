package beanq

import (
	"fmt"
	"strings"

	"github.com/retail-ai-inc/beanq/v4/internal/driver/btls"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func mongoClientOptions(config *Mongo) (*options.ClientOptions, error) {
	if config == nil {
		return nil, ErrInvalidConfig.WithMessage("mongo config is nil")
	}

	opts := options.Client().ApplyURI(mongoURI(config.Host, config.Port)).
		SetConnectTimeout(config.ConnectTimeOut).
		SetMaxPoolSize(config.MaxConnectionPoolSize).
		SetMaxConnIdleTime(config.MaxConnectionLifeTime)

	if credential, ok := mongoCredential(config); ok {
		opts.SetAuth(credential)
	}

	if config.SSL.On {
		tlsConfig, err := btls.LoadTLSConfigFromCA(config.SSL.CAFile, config.SSL.Verify)
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(tlsConfig)
	}

	return opts, nil
}

func mongoURI(host, port string) string {
	return fmt.Sprintf("mongodb://%s:%s", host, normalizePort(port))
}

func normalizePort(port string) string {
	return strings.TrimLeft(port, ":")
}

func mongoCredential(config *Mongo) (options.Credential, bool) {
	if config.UserName == "" || config.Password == "" {
		return options.Credential{}, false
	}
	return options.Credential{
		AuthSource: config.Database,
		Username:   config.UserName,
		Password:   config.Password,
	}, true
}
