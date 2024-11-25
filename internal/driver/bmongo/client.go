package bmongo

import (
	"context"
	"fmt"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
	"time"
)

var ()

type MongoLog struct {
	database   *mongo.Database
	collection string
}

func NewMongoLog(ctx context.Context,

	host, port string,
	connectTimeOut, maxConnectionLifeTime time.Duration,
	maxConnectionPoolSize uint64,
	database, collection, userName, password string,
) *MongoLog {

	uri := strings.Join([]string{"mongodb://", host, port}, "")

	if uri == "" {
		return nil
	}

	opts := options.Client().ApplyURI(uri).
		SetConnectTimeout(connectTimeOut).
		SetMaxConnIdleTime(maxConnectionLifeTime).
		SetMaxPoolSize(maxConnectionPoolSize)

	if userName != "" && password != "" {
		auth := options.Credential{
			AuthSource: database,
			Username:   userName,
			Password:   password,
		}
		opts.SetAuth(auth)
	}

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		logger.New().Fatal(err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		logger.New().Fatal(err)
	}

	return &MongoLog{
		database:   client.Database(database),
		collection: collection,
	}
}

func (t *MongoLog) Migrate(ctx context.Context, data []map[string]any) error {

	datas := make(bson.A, 0, len(data))
	for _, v := range data {
		datas = append(datas, bson.M(v))
	}

	if _, err := t.database.Collection(t.collection).InsertMany(ctx, datas); err != nil {
		return fmt.Errorf("Mongo Error:%w \n", err)
	}
	return nil
}
