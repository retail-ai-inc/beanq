package beanq

import (
	"context"

	"github.com/retail-ai-inc/beanq/v4/internal/driver/bmongo"
)

func newMongoStore(ctx context.Context, mongo *Mongo) (*bmongo.MongoStore, error) {
	return bmongo.NewMongoStore(ctx, mongo.mongoStoreConfig("event"))
}

func (mongo *Mongo) mongoStoreConfig(collection string) bmongo.Config {
	return bmongo.Config{
		Host: mongo.Host, Port: mongo.Port, Database: mongo.Database,
		Collection: mongo.collectionName(collection, ""),
		Username:   mongo.UserName, Password: mongo.Password,
		ConnectTimeout: mongo.ConnectTimeOut, MaxConnIdleTime: mongo.MaxConnectionLifeTime,
		MaxPoolSize: mongo.MaxConnectionPoolSize,
		TLSOn:       mongo.SSL.On, CAFile: mongo.SSL.CAFile,
		VerifyCertificate: mongo.SSL.Verify, HotReload: mongo.SSL.HotReload,
	}
}
