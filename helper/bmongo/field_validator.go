package bmongo

import (
	"context"
	"time"

	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	eventLogValid = bson.D{
		{"$jsonSchema", bson.D{
			{"bsonType", "object"},
			{"properties", bson.D{
				{"id", bson.D{
					{"bsonType", "string"},
					{"description", "unique id"},
				}},
				{"topic", bson.D{
					{"bsonType", "string"},
					{"description", ""},
				}},
				{"channel", bson.D{
					{"bsonType", "string"},
					{"description", ""},
				}},
				{"logType", bson.D{
					{"bsonType", "string"},
				}},
				{"moodType", bson.D{
					{"bsonType", "string"},
				}},
				{"status", bson.D{
					{"bsonType", "string"},
				}},
				{"maxLen", bson.D{
					{"bsonType", "long"},
				}},
				{"timeToRun", bson.D{
					{"bsonType", "long"},
				}},
				{"retry", bson.D{
					{"bsonType", "long"},
				}},
				{"addTime", bson.D{
					{"bsonType", "string"},
				}},
				{"executeTime", bson.D{
					{"bsonType", "string"},
				}},
				{"payload", bson.D{
					{"bsonType", "string"},
				}},
				{"priority", bson.D{
					{"bsonType", "long"},
				}},
			}},
		}},
	}
	optLogValid = bson.D{
		{"$jsonSchema", bson.D{
			{"bsonType", "object"},
			{"properties", bson.D{
				{"uri", bson.D{
					{"bsonType", "string"},
				}},
				{"addTime", bson.D{
					{"bsonType", "string"},
				}},
				{"data", bson.D{
					{"bsonType", "string"},
				}},
				{"logType", bson.D{
					{"bsonType", "string"},
				}},
				{"user", bson.D{
					{"bsonType", "string"},
				}},
				{"expireAt", bson.D{
					{"bsonType", "string"},
				}},
			}},
		}},
	}
	managerValid = bson.D{
		{"$jsonSchema", bson.D{
			{"bsonType", "object"},
			{"properties", bson.D{
				{"account", bson.D{
					{"bsonType", "string"},
				}},
				{"password", bson.D{
					{"bsonType", "string"},
				}},
				{"type", bson.D{
					{"bsonType", "string"},
				}},
				{"active", bson.D{
					{"bsonType", "long"},
				}},
				{"detail", bson.D{
					{"bsonType", "string"},
				}},
				{"roleId", bson.D{
					{"bsonType", "string"},
				}},
				{"createAt", bson.D{
					{"bsonType", "date"},
				}},
				{"updateAt", bson.D{
					{"bsonType", "date"},
				}},
			}},
		}},
	}
	roleValid = bson.D{
		{"$jsonSchema", bson.D{
			{"bsonType", "object"},
			{"properties", bson.D{
				{"name", bson.D{
					{"bsonType", "string"},
				}},
				{"roles", bson.D{
					{"bsonType", "array"},
				}},
				{"createAt", bson.D{
					{"bsonType", "date"},
				}},
				{"updateAt", bson.D{
					{"bsonType", "date"},
				}},
			}},
		}},
	}
	workflowValid = bson.D{
		{"$jsonSchema", bson.D{
			{"bsonType", "object"},
			{"properties", bson.D{
				{"Channel", bson.D{
					{"bsonType", "string"},
				}},
				{"Topic", bson.D{
					{"bsonType", "string"},
				}},
				{"Gid", bson.D{
					{"bsonType", "string"},
				}},
				{"MessageId", bson.D{
					{"bsonType", "string"},
				}},
				{"Statement", bson.D{
					{"bsonType", "string"},
				}},
				{"Status", bson.D{
					{"bsonType", "string"},
				}},
				{"TaskId", bson.D{
					{"bsonType", "string"},
				}},
				{"CreatedAt", bson.D{
					{"bsonType", "date"},
				}},
				{"UpdatedAt", bson.D{
					{"bsonType", "date"},
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
		{"listCollections", 1},
		{"filter", bson.D{{"name", t}}},
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
			continue
		}
		if v, ok := data["name"]; ok {
			if v.(string) == string(t) {
				isExist = true
				break
			}
		}
		data = nil
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
			Keys:    bson.D{{key, sort}},
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
			Keys:    bson.D{{key, 1}},
			Options: options.Index().SetExpireAfterSeconds(cast.ToInt32(duration.Seconds())),
		}); err != nil {
			return err
		}
	}
	return nil
}
