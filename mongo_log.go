package beanq

import (
	"context"
	"fmt"
	"strings"

	"github.com/retail-ai-inc/beanq/helper/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoLog struct {
	database   *mongo.Database
	collection string
}

func NewMongoLog(ctx context.Context, config *BeanqConfig) *MongoLog {

	historyCfg := config.History

	uri := ""
	if historyCfg.On {
		uri = strings.Join([]string{"mongodb://", historyCfg.Mongo.Host, historyCfg.Mongo.Port}, "")
	}
	if uri == "" {
		return nil
	}

	opts := options.Client().ApplyURI(uri).
		SetConnectTimeout(historyCfg.Mongo.ConnectTimeOut).
		SetMaxPoolSize(historyCfg.Mongo.MaxConnectionPoolSize).
		SetMaxConnIdleTime(historyCfg.Mongo.MaxConnectionLifeTime)
	if historyCfg.Mongo.UserName != "" && historyCfg.Mongo.Password != "" {
		auth := options.Credential{
			AuthSource: historyCfg.Mongo.Database,
			Username:   historyCfg.Mongo.UserName,
			Password:   historyCfg.Mongo.Password,
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
		database:   client.Database(historyCfg.Mongo.Database),
		collection: historyCfg.Mongo.Collection,
	}
}

// Archive save log
func (t *MongoLog) Archive(ctx context.Context, result *Message, isSequential bool) error {
	return nil
}

// Obsolete log
// If you don't want to implement an elimination strategy, you can skip implementing the method
func (t *MongoLog) Obsolete(ctx context.Context, data []map[string]any) error {

	datas := make(bson.A, 0, len(data))
	for _, v := range data {
		datas = append(datas, bson.M(v))
	}
	if _, err := t.database.Collection(t.collection).InsertMany(ctx, datas); err != nil {
		return fmt.Errorf("Mongo Error:%w \n", err)
	}
	return nil
}
