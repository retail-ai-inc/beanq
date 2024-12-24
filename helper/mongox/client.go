package mongox

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"strings"
	"sync"
	"time"
)

var (
	mongoOnce sync.Once
	mongoX    *MongoX
)

type MongoX struct {
	database   *mongo.Database
	collection string
}

func NewMongo(host, port string, username, password string, database, collection string, connectTimeOut time.Duration, maxConnectionPoolSize uint64, maxConnectionLifeTime time.Duration) *MongoX {
	mongoOnce.Do(func() {

		uri := strings.Join([]string{"mongodb://", host, port}, "")

		opts := options.Client().ApplyURI(uri).
			SetConnectTimeout(connectTimeOut).
			SetMaxPoolSize(maxConnectionPoolSize).
			SetMaxConnIdleTime(maxConnectionLifeTime)
		if username != "" && password != "" {
			auth := options.Credential{
				AuthSource: database,
				Username:   username,
				Password:   password,
			}
			opts.SetAuth(auth)
		}

		ctx := context.Background()
		client, err := mongo.Connect(ctx, opts)
		if err != nil {
			log.Fatal(err)
		}
		if err := client.Ping(ctx, nil); err != nil {
			log.Fatal(err)
		}
		mongoX = &MongoX{
			database:   client.Database(database),
			collection: collection,
		}
	})
	return mongoX
}

func (t *MongoX) EventLogs(ctx context.Context, filter bson.M, page, pageSize int64) ([]bson.M, int64, error) {
	skip := (page - 1) * pageSize
	if skip < 0 {
		skip = 0
	}
	opts := options.Find()
	opts.SetSkip(skip)
	opts.SetLimit(pageSize)
	opts.SetSort(bson.D{{Key: "addTime", Value: 1}})

	cursor, err := t.database.Collection(t.collection).Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	var data []bson.M
	if err := cursor.All(ctx, &data); err != nil {
		return nil, 0, err
	}
	total, err := t.database.Collection(t.collection).CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (t *MongoX) DetailEventLog(ctx context.Context, id string) (bson.M, error) {

	filter := bson.M{}
	if id != "" {
		if _, err := primitive.ObjectIDFromHex(id); err != nil {
			return nil, err
		}
		filter["id"] = id
	}

	single := t.database.Collection(t.collection).FindOne(ctx, filter)
	if err := single.Err(); err != nil {
		return nil, err
	}
	var data bson.M
	if err := single.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func (t *MongoX) Delete(ctx context.Context, id string) (int64, error) {
	filter := bson.M{}
	if id != "" {
		nid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return 0, err
		}
		filter["_id"] = nid
	}
	result, err := t.database.Collection(t.collection).DeleteOne(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

func (t *MongoX) Edit(ctx context.Context, id string, payload any) (int64, error) {
	filter := bson.M{}
	if id != "" {
		nid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return 0, err
		}
		filter["_id"] = nid
	}
	update := bson.D{
		{Key: "$set", Value: bson.D{{Key: "payload", Value: payload}}},
	}
	result, err := t.database.Collection(t.collection).UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}
