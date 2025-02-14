package bmongo

import (
	"context"
	"errors"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"math"
	"strings"
	"sync"
	"time"
)

var (
	mongoOnce sync.Once
	mgo       *BMongo
)

type BMongo struct {
	database           *mongo.Database
	eventCollection    string
	workflowCollection string
	managerCollection  string
	optCollection      string
	roleCollection     string
}

func NewMongo(host, port string,
	username, password string,
	database string,
	collections map[string]string,
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
			database:           client.Database(database),
			eventCollection:    "event_logs",
			workflowCollection: "workflow_logs",
			managerCollection:  "managers",
			optCollection:      "opt_logs",
			roleCollection:     "roles",
		}
		if v, ok := collections["event"]; ok {
			mgo.eventCollection = v
		}
		if v, ok := collections["workflow"]; ok {
			mgo.workflowCollection = v
		}
		if v, ok := collections["manager"]; ok {
			mgo.managerCollection = v
		}
		if v, ok := collections["opt"]; ok {
			mgo.optCollection = v
		}
		if v, ok := collections["roles"]; ok {
			mgo.roleCollection = v
		}
	})
	return mgo
}

func (t *BMongo) DocumentCount(ctx context.Context, status string) (int64, error) {

	filter := bson.M{}
	filter["status"] = status
	total, err := t.database.Collection(t.eventCollection).CountDocuments(ctx, filter)
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

	cursor, err := t.database.Collection(t.workflowCollection).Find(ctx, filter, opts)
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
	total, err := t.database.Collection(t.workflowCollection).CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (t *BMongo) EventLogs(ctx context.Context, filter bson.M, page, pageSize int64) ([]bson.M, float64, error) {
	skip := (page - 1) * pageSize
	if skip < 0 {
		skip = 0
	}
	opts := options.Find()
	opts.SetSkip(skip)
	opts.SetLimit(pageSize)
	opts.SetSort(bson.D{{Key: "addTime", Value: 1}})

	cursor, err := t.database.Collection(t.eventCollection).Find(ctx, filter, opts)
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
	total, err := t.database.Collection(t.eventCollection).CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return data, math.Ceil(float64(total) / float64(pageSize)), nil
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

	single := t.database.Collection(t.eventCollection).FindOne(ctx, filter)
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
	result, err := t.database.Collection(t.eventCollection).DeleteOne(ctx, filter)
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
	result, err := t.database.Collection(t.eventCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func (t *BMongo) AddOptLog(ctx context.Context, data map[string]any) error {

	_, err := t.database.Collection(t.optCollection).InsertOne(ctx, data)
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

	cursor, err := t.database.Collection(t.optCollection).Find(ctx, filter, opts)
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
	total, err := t.database.Collection(t.optCollection).CountDocuments(ctx, filter)
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
	result, err := t.database.Collection(t.optCollection).DeleteOne(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

type User struct {
	CreateAt time.Time `bson:"createAt" json:"createAt"`
	UpdateAt time.Time `bson:"updateAt" json:"updateAt"`
	Account  string    `bson:"account" json:"account"`
	Password string    `bson:"password" json:"password"`
	Type     string    `bson:"type" json:"type"`
	Detail   string    `bson:"detail" json:"detail"`
	Active   int32     `bson:"active" json:"active"`
	RoleId   string    `bson:"roleId" json:"roleId"`
}

func (t *BMongo) AddUser(ctx context.Context, user *User) error {

	user.CreateAt = time.Now()
	_, err := t.database.Collection(t.managerCollection).InsertOne(ctx, user)
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
	result, err := t.database.Collection(t.managerCollection).DeleteOne(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

func (t *BMongo) CheckUser(ctx context.Context, account, password string) (*User, error) {

	filter := bson.M{
		"account":  account,
		"password": password,
		"active":   1,
	}

	var user User
	result := t.database.Collection(t.managerCollection).FindOne(ctx, filter)
	if err := result.Err(); err != nil {
		if errors.Is(err, mongo.ErrNilDocument) {
			return nil, nil
		}
		return nil, err
	}
	if err := result.Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (t *BMongo) CheckGoogleUser(ctx context.Context, account string) (*User, error) {
	filter := bson.M{
		"account": account,
		"type":    "google",
		"active":  1,
	}

	var user User
	result := t.database.Collection(t.managerCollection).FindOne(ctx, filter)
	if err := result.Err(); err != nil {
		if errors.Is(err, mongo.ErrNilDocument) {
			return nil, nil
		}
		return nil, err
	}
	if err := result.Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
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

	result, err := t.database.Collection(t.managerCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func (t *BMongo) UserLogs(ctx context.Context, filter bson.M, page, pageSize int64) ([]bson.M, float64, error) {
	skip := (page - 1) * pageSize
	if skip < 0 {
		skip = 0
	}
	opts := options.Find()
	opts.SetSkip(skip)
	opts.SetLimit(pageSize)
	opts.SetSort(bson.D{{Key: "addTime", Value: 1}})

	cursor, err := t.database.Collection(t.managerCollection).Find(ctx, filter, opts)
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
	total, err := t.database.Collection(t.managerCollection).CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return data, math.Ceil(float64(total) / float64(pageSize)), nil
}

type Role struct {
	CreateAt time.Time `bson:"createAt" json:"createAt"`
	UpdateAt time.Time `bson:"updateAt" json:"updateAt"`
	Name     string    `bson:"name" json:"name"`
	Roles    []int     `bson:"roles" json:"roles"`
}

func (t *BMongo) Roles(ctx context.Context, m bson.M, page, pageSize int64) ([]bson.M, float64, error) {
	skip := (page - 1) * pageSize
	if skip < 0 {
		skip = 0
	}
	opts := options.Find()
	opts.SetSkip(skip)
	opts.SetLimit(pageSize)
	opts.SetSort(bson.D{{Key: "createAt", Value: 1}})

	cursor, err := t.database.Collection(t.roleCollection).Find(ctx, m, opts)
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
	total, err := t.database.Collection(t.roleCollection).CountDocuments(ctx, m)
	if err != nil {
		return nil, 0, err
	}
	return data, math.Ceil(float64(total) / float64(pageSize)), nil
}

func (t *BMongo) AddRole(ctx context.Context, role *Role) error {

	role.CreateAt = time.Now()
	_, err := t.database.Collection(t.roleCollection).InsertOne(ctx, role)
	return err
}

func (t *BMongo) DeleteRole(ctx context.Context, id string) (int64, error) {

	filter := bson.M{}
	if id != "" {
		nid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return 0, err
		}
		filter["_id"] = nid
	}
	result, err := t.database.Collection(t.roleCollection).DeleteOne(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

func (t *BMongo) EditRole(ctx context.Context, id string, data map[string]any) (int64, error) {
	filter := bson.M{}
	if id != "" {
		nid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return 0, err
		}
		filter["_id"] = nid
	}

	var values bson.D

	if v, ok := data["roles"]; ok {
		values = append(values, bson.E{Key: "roles", Value: v})
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

	result, err := t.database.Collection(t.roleCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}
