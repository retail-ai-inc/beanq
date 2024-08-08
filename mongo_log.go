package beanq

import (
	"context"
	"strings"
	"time"

	"github.com/retail-ai-inc/beanq/helper/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoLog struct {
	database *mongo.Database
}

const MongoCollection string = "logs"

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
		database: client.Database(historyCfg.Mongo.Database),
	}
}

// Archive save log
func (t *MongoLog) Archive(ctx context.Context, result *Message) error {
	data := bson.M{
		"sid":       result.Id,
		"status":    result.Status,
		"level":     result.Level,
		"type":      result.Info,
		"data":      result,
		"createdAt": time.Now(),
		"updatedAt": time.Now(),
	}
	if _, err := t.database.Collection(MongoCollection).InsertOne(ctx, data); err != nil {
		return err
	}
	return nil
}

// Obsolete log
// If you don't want to implement an elimination strategy, you can skip implementing the method
func (t *MongoLog) Obsolete(ctx context.Context) {

}
