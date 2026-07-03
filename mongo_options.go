package beanq

import (
	"fmt"
	"strings"

	"github.com/retail-ai-inc/beanq/v4/internal/driver/btls"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func mongoClientOptions(config *Mongo) (*options.ClientOptions, error) {
	if config == nil {
		return nil, fmt.Errorf("mongo config is nil")
	}

	port := strings.TrimLeft(config.Port, ":")
	port = fmt.Sprintf(":%s", port)
	uri := strings.Join([]string{"mongodb://", config.Host, port}, "")

	opts := options.Client().ApplyURI(uri).
		SetConnectTimeout(config.ConnectTimeOut).
		SetMaxPoolSize(config.MaxConnectionPoolSize).
		SetMaxConnIdleTime(config.MaxConnectionLifeTime)

	if config.UserName != "" && config.Password != "" {
		opts.SetAuth(options.Credential{
			AuthSource: config.Database,
			Username:   config.UserName,
			Password:   config.Password,
		})
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
