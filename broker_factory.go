package beanq

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	bmongo2 "github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/bredis"
)

type brokerComponents struct {
	queue         queueGateway
	locker        locker
	status        statusReader
	client        any
	strategy      brokerStrategy
	tool          *bredis.UITool
	captureConfig *capture.Config
}

type brokerBuilder func(config *BeanqConfig) (*brokerComponents, error)

var defaultBrokerBuilder brokerBuilder = buildBrokerComponents

func buildBrokerComponents(config *BeanqConfig) (*brokerComponents, error) {
	switch config.Broker {
	case "redis":
		return buildRedisBrokerComponents(config)
	default:
		return nil, ErrUnsupportedBroker.WithMessage(config.Broker)
	}
}

func buildRedisBrokerComponents(config *BeanqConfig) (*brokerComponents, error) {
	cfg := config.Redis
	client, err := bredis.NewRdb(cfg.IsCluster, cfg.Host, cfg.Port, cfg.Username,
		cfg.Password, cfg.Database,
		cfg.MaxRetries, cfg.DialTimeout, cfg.ReadTimeout, cfg.WriteTimeout, cfg.PoolTimeout, cfg.PoolSize, cfg.MinIdleConnections,
		cfg.SSL.On, cfg.SSL.CAFile, cfg.SSL.Verify, cfg.SSL.HotReload)
	if err != nil {
		return nil, err
	}

	rdbBroker := bredis.NewBroker(client, cfg.Prefix, cfg.MaxLen, config.MinConsumers, config.ConsumerPoolSize, config.DeadLetterIdleTime)
	components := &brokerComponents{
		queue:  rdbBroker,
		locker: rdbBroker,
		status: bredis.NewStatus(client, cfg.Prefix),
		client: client,
		strategy: func(moodType btype.MoodType, config *capture.Config) queueGateway {
			return rdbBroker.Mood(moodType, config)
		},
		tool: bredis.NewUITool(client, cfg.Prefix),
	}

	if config.History.On {
		components.captureConfig = captureConfigFromMongo(config.Mongo)
	}
	return components, nil
}

func captureConfigFromMongo(mcfg *Mongo) *capture.Config {
	if mcfg == nil {
		return nil
	}

	collections := make(map[string]string, len(mcfg.Collections))
	for key, collection := range mcfg.Collections {
		collections[key] = collection.Name
	}

	nmgo := bmongo2.NewMongo(mcfg.Host,
		mcfg.Port, mcfg.UserName,
		mcfg.Password,
		mcfg.Database,
		collections,
		mcfg.ConnectTimeOut,
		mcfg.MaxConnectionPoolSize,
		mcfg.MaxConnectionLifeTime,
		bmongo2.MongoSSLConfig{
			On:     mcfg.SSL.On,
			CAFile: mcfg.SSL.CAFile,
			Verify: mcfg.SSL.Verify,
		})

	return getConfig(nmgo)
}

func newRedisMigrateLog(ctx context.Context, config *BeanqConfig, client redis.UniversalClient) MigrationRunner {
	var migrate MigrationRunner
	if config.History.On && config.Mongo != nil {
		migrate = newMongoStore(ctx, config.Mongo)
	}
	return bredis.NewLog(client, config.Redis.Prefix, migrate)
}

func logBrokerBuildFailure(err error, broker string) {
	if err != nil {
		logger.New().Panic("new broker err:", err)
	}
	if broker == "" {
		logger.New().Panic("not support broker type:", broker)
	}
}

func configLookupTimeout() time.Duration {
	return 10 * time.Second
}
