package bmongo

import (
	"context"
	"time"

	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	eventLogValid = bson.D{
		{Key: "$jsonSchema", Value: bson.D{
			{Key: "bsonType", Value: "object"},
			{Key: "properties", Value: bson.D{
				{Key: "id", Value: bson.D{
					{Key: "bsonType", Value: "string"},
					{Key: "description", Value: "unique id"},
				}},
				{Key: "topic", Value: bson.D{
					{Key: "bsonType", Value: "string"},
					{Key: "description", Value: ""},
				}},
				{Key: "channel", Value: bson.D{
					{Key: "bsonType", Value: "string"},
					{Key: "description", Value: ""},
				}},
				{Key: "logType", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "moodType", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "status", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "maxLen", Value: bson.D{
					{Key: "bsonType", Value: "long"},
				}},
				{Key: "timeToRun", Value: bson.D{
					{Key: "bsonType", Value: "long"},
				}},
				{Key: "retry", Value: bson.D{
					{Key: "bsonType", Value: "long"},
				}},
				{Key: "addTime", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "executeTime", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "payload", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "priority", Value: bson.D{
					{Key: "bsonType", Value: "long"},
				}},
			}},
		}},
	}
	optLogValid = bson.D{
		{Key: "$jsonSchema", Value: bson.D{
			{Key: "bsonType", Value: "object"},
			{Key: "properties", Value: bson.D{
				{Key: "uri", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "addTime", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "data", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "logType", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "user", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "expireAt", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
			}},
		}},
	}
	managerValid = bson.D{
		{Key: "$jsonSchema", Value: bson.D{
			{Key: "bsonType", Value: "object"},
			{Key: "properties", Value: bson.D{
				{Key: "account", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "password", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "type", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "active", Value: bson.D{
					{Key: "bsonType", Value: "long"},
				}},
				{Key: "detail", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "roleId", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "createAt", Value: bson.D{
					{Key: "bsonType", Value: "date"},
				}},
				{Key: "updateAt", Value: bson.D{
					{Key: "bsonType", Value: "date"},
				}},
			}},
		}},
	}
	roleValid = bson.D{
		{Key: "$jsonSchema", Value: bson.D{
			{Key: "bsonType", Value: "object"},
			{Key: "properties", Value: bson.D{
				{Key: "name", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "roles", Value: bson.D{
					{Key: "bsonType", Value: "array"},
				}},
				{Key: "createAt", Value: bson.D{
					{Key: "bsonType", Value: "date"},
				}},
				{Key: "updateAt", Value: bson.D{
					{Key: "bsonType", Value: "date"},
				}},
			}},
		}},
	}
	workflowValid = bson.D{
		{Key: "$jsonSchema", Value: bson.D{
			{Key: "bsonType", Value: "object"},
			{Key: "properties", Value: bson.D{
				{Key: "Channel", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "Topic", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "Gid", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "MessageId", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "Statement", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "Status", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "TaskId", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "CreatedAt", Value: bson.D{
					{Key: "bsonType", Value: "date"},
				}},
				{Key: "UpdatedAt", Value: bson.D{
					{Key: "bsonType", Value: "date"},
				}},
			}},
		}},
	}
)

type CollectionType string

const (
	EventType    CollectionType = "event"
	ManagerType  CollectionType = "manager"
	OptType      CollectionType = "opt"
	RoleType     CollectionType = "role"
	WorkFLowType CollectionType = "workflow"
)

type Collection string

// Create collection
func (t Collection) Create(ctx context.Context, database *mongo.Database, tp CollectionType) error {

	cursor, err := database.RunCommandCursor(ctx, bson.D{
		{Key: "listCollections", Value: 1},
		{Key: "filter", Value: bson.D{{Key: "name", Value: t}}},
	})
	if err != nil {
		return err
	}

	defer func() {
		_ = cursor.Close(ctx)
	}()

	isExist := false
	data := make(map[string]any, 5)
	for cursor.Next(ctx) {
		if err := cursor.Decode(&data); err != nil {
			logger.New().Error(err)
			continue
		}
		if v, ok := data["name"]; ok && v.(string) == string(t) {
			isExist = true
			break
		}
	}
	if !isExist {
		opts := options.CreateCollection()

		switch tp {
		case OptType:
			opts.SetValidator(optLogValid)
			opts.SetValidationLevel("strict")
		case EventType:
			opts.SetValidator(eventLogValid)
			opts.SetValidationLevel("strict")
		case ManagerType:
			opts.SetValidator(managerValid)
			opts.SetValidationLevel("strict")
		case RoleType:
			opts.SetValidator(roleValid)
			opts.SetValidationLevel("strict")
		case WorkFLowType:
			opts.SetValidator(workflowValid)
			opts.SetValidationLevel("strict")
		}

		if err := database.CreateCollection(ctx, string(t), opts); err != nil {
			return err
		}
	}

	return nil

}

// CreateIndex create normal index
func (t Collection) CreateIndex(ctx context.Context, database *mongo.Database, key string, sort int) error {

	// indexs list
	cursor, err := database.Collection(string(t)).Indexes().List(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()
	data := make(map[string]any, 3)
	isExist := false
	for cursor.Next(ctx) {
		if err := cursor.Decode(&data); err != nil {
			logger.New().Error(err)
			continue
		}
		if v, ok := data["key"]; ok {
			if _, keyok := v.(map[string]any)[key]; keyok {
				isExist = true
				break
			}
		}
	}
	if !isExist {
		if _, err := database.Collection(string(t)).Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: key, Value: sort}},
			Options: nil,
		}); err != nil {
			return err
		}
	}
	return nil
}

// CreateTTLIndex create ttl index
func (t Collection) CreateTTLIndex(ctx context.Context, database *mongo.Database, duration time.Duration) error {

	key := "expireAt"
	cursor, err := database.Collection(string(t)).Indexes().List(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()
	data := make(map[string]any, 3)
	isExist := false
	for cursor.Next(ctx) {
		if err := cursor.Decode(&data); err != nil {
			continue
		}
		if v, ok := data["key"]; ok {
			if _, keyok := v.(map[string]any)[key]; keyok {
				isExist = true
				break
			}
		}
	}
	if !isExist {
		if _, err := database.Collection(string(t)).Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: key, Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(cast.ToInt32(duration.Seconds())),
		}); err != nil {
			return err
		}
	}
	return nil
}
