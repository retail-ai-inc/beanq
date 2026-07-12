package beanq

import (
	"context"

	"github.com/retail-ai-inc/beanq/v4/internal/driver/bmongo"
)

func newMongoStore(ctx context.Context, mongo *Mongo) *bmongo.MongoStore {
	return bmongo.NewMongoStore(ctx,
		mongo.Host,
		mongo.Port,
		mongo.ConnectTimeOut,
		mongo.MaxConnectionLifeTime,
		mongo.MaxConnectionPoolSize,
		mongo.Database,
		mongo.Collections["event"].Name,
		mongo.UserName,
		mongo.Password,
		mongo.SSL.On,
		mongo.SSL.CAFile,
		mongo.SSL.Verify,
		mongo.SSL.HotReload)
}
