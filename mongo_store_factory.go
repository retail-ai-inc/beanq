package beanq

import (
	"context"

	"github.com/retail-ai-inc/beanq/v4/internal/driver/bmongo"
)

func newMongoStore(ctx context.Context, mongo *Mongo) (*bmongo.MongoStore, error) {
	return bmongo.NewMongoStore(ctx, bmongo.Config{
		Host: mongo.Host, Port: mongo.Port, Database: mongo.Database,
		Collection: mongo.Collections["event"].Name,
		Username:   mongo.UserName, Password: mongo.Password,
		ConnectTimeout: mongo.ConnectTimeOut, MaxConnIdleTime: mongo.MaxConnectionLifeTime,
		MaxPoolSize: mongo.MaxConnectionPoolSize,
		TLSOn:       mongo.SSL.On, CAFile: mongo.SSL.CAFile,
		VerifyCertificate: mongo.SSL.Verify, HotReload: mongo.SSL.HotReload,
	})
}
