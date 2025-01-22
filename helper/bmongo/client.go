package bmongo

import (
	"context"
	"github.com/spf13/cast"
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
	mgo       *BMongo
)

const (
	manager     = "managers"
	optLog      = "opt_logs"
	workflowLog = "workflow_logs"
)

type BMongo struct {
	database   *mongo.Database
	collection string
}

func NewMongo(host, port string,
	username, password string,
	database, collection string,
	connectTimeOut time.Duration, maxConnectionPoolSize uint64,
	maxConnectionLifeTime time.Duration) *BMongo {
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
		mgo = &BMongo{
			database:   client.Database(database),
			collection: collection,
		}
	})
	return mgo
}

func (t *BMongo) DocumentCount(ctx context.Context, status string) (int64, error) {

	filter := bson.M{}
	filter["status"] = status
	total, err := t.database.Collection("event_logs").CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (t *BMongo) WorkFlowLogs(ctx context.Context, filter bson.M, page, pageSize int64) ([]bson.M, int64, error) {
	skip := (page - 1) * pageSize
	if skip < 0 {
		skip = 0
	}
	opts := options.Find()
	opts.SetSkip(skip)
	opts.SetLimit(pageSize)
	opts.SetSort(bson.D{{Key: "CreatedAt", Value: 1}})

	cursor, err := t.database.Collection(workflowLog).Find(ctx, filter, opts)
	defer func() {
		_ = cursor.Close(ctx)
	}()
	if err != nil {
		return nil, 0, err
	}
	var data []bson.M
	if err := cursor.All(ctx, &data); err != nil {
		return nil, 0, err
	}
	total, err := t.database.Collection(workflowLog).CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (t *BMongo) EventLogs(ctx context.Context, filter bson.M, page, pageSize int64) ([]bson.M, int64, error) {
	skip := (page - 1) * pageSize
	if skip < 0 {
		skip = 0
	}
	opts := options.Find()
	opts.SetSkip(skip)
	opts.SetLimit(pageSize)
	opts.SetSort(bson.D{{Key: "addTime", Value: 1}})

	cursor, err := t.database.Collection(t.collection).Find(ctx, filter, opts)
	defer func() {
		_ = cursor.Close(ctx)
	}()
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

func (t *BMongo) DetailEventLog(ctx context.Context, id string) (bson.M, error) {

	filter := bson.M{}
	if id != "" {
		if objectId, err := primitive.ObjectIDFromHex(id); err != nil {
			return nil, err
		} else {
			filter["_id"] = objectId
		}
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

func (t *BMongo) Delete(ctx context.Context, id string) (int64, error) {
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

func (t *BMongo) Edit(ctx context.Context, id string, payload any) (int64, error) {
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

func (t *BMongo) AddOptLog(ctx context.Context, data map[string]any) error {

	_, err := t.database.Collection(optLog).InsertOne(ctx, data)
	return err
}

func (t *BMongo) OptLogs(ctx context.Context, page, pageSize int64) ([]bson.M, int64, error) {

	skip := (page - 1) * pageSize
	if skip < 0 {
		skip = 0
	}
	opts := options.Find()
	opts.SetSkip(skip)
	opts.SetLimit(pageSize)
	opts.SetSort(bson.D{{Key: "addTime", Value: 1}})

	filter := bson.M{}

	cursor, err := t.database.Collection(optLog).Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()
	var data []bson.M
	if err := cursor.All(ctx, &data); err != nil {
		return nil, 0, err
	}
	total, err := t.database.Collection(optLog).CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (t *BMongo) DeleteOptLog(ctx context.Context, id string) (int64, error) {
	filter := bson.M{}
	if id != "" {
		nid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return 0, err
		}
		filter["_id"] = nid
	}
	result, err := t.database.Collection(optLog).DeleteOne(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

func (t *BMongo) AddUser(ctx context.Context, data map[string]any) error {

	_, err := t.database.Collection(manager).InsertOne(ctx, data)
	return err
}

func (t *BMongo) DeleteUser(ctx context.Context, id string) (int64, error) {

	filter := bson.M{}
	if id != "" {
		nid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return 0, err
		}
		filter["_id"] = nid
	}
	result, err := t.database.Collection(manager).DeleteOne(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

func (t *BMongo) EditUser(ctx context.Context, id string, data map[string]any) (int64, error) {
	filter := bson.M{}
	if id != "" {
		nid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return 0, err
		}
		filter["_id"] = nid
	}

	var values bson.D
	if v, ok := data["password"]; ok {
		if cast.ToString(v) != "" {
			values = append(values, bson.E{Key: "password", Value: v})
		}
	}
	if v, ok := data["type"]; ok {
		if cast.ToString(v) != "" {
			values = append(values, bson.E{Key: "type", Value: v})
		}
	}
	if v, ok := data["active"]; ok {
		values = append(values, bson.E{Key: "active", Value: v})
	}
	if v, ok := data["detail"]; ok {
		if cast.ToString(v) != "" {
			values = append(values, bson.E{Key: "detail", Value: v})
		}
	}
	values = append(values, bson.E{Key: "updateAt", Value: time.Now()})

	update := bson.D{
		{Key: "$set", Value: values},
	}

	result, err := t.database.Collection(manager).UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func (t *BMongo) UserLogs(ctx context.Context, filter bson.M, page, pageSize int64) ([]bson.M, int64, error) {
	skip := (page - 1) * pageSize
	if skip < 0 {
		skip = 0
	}
	opts := options.Find()
	opts.SetSkip(skip)
	opts.SetLimit(pageSize)
	opts.SetSort(bson.D{{Key: "addTime", Value: 1}})

	cursor, err := t.database.Collection(manager).Find(ctx, filter, opts)
	defer func() {
		_ = cursor.Close(ctx)
	}()
	if err != nil {
		return nil, 0, err
	}
	var data []bson.M
	if err := cursor.All(ctx, &data); err != nil {
		return nil, 0, err
	}
	total, err := t.database.Collection(manager).CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return data, total, nil
}
