package bmongo

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Define the validator for different types of collections
var (
	validators = map[CollectionType]bson.D{
		OptType: bson.D{
			{Key: "$jsonSchema", Value: bson.D{
				{Key: "bsonType", Value: "object"},
				{Key: "properties", Value: bson.D{
					{Key: "uri", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "addTime", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "data", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "logType", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "user", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "expireAt", Value: bson.D{{Key: "bsonType", Value: "string"}}},
				}},
			}},
		},
		EventType: bson.D{
			{Key: "$jsonSchema", Value: bson.D{
				{Key: "bsonType", Value: "object"},
				{Key: "properties", Value: bson.D{
					{Key: "id", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "topic", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "channel", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "logType", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "moodType", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "status", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "maxLen", Value: bson.D{{Key: "bsonType", Value: "long"}}},
					{Key: "timeToRun", Value: bson.D{{Key: "bsonType", Value: "long"}}},
					{Key: "retry", Value: bson.D{{Key: "bsonType", Value: "long"}}},
					{Key: "addTime", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "executeTime", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "payload", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "priority", Value: bson.D{{Key: "bsonType", Value: "long"}}},
				}},
			}},
		},
		ManagerType: bson.D{
			{Key: "$jsonSchema", Value: bson.D{
				{Key: "bsonType", Value: "object"},
				{Key: "properties", Value: bson.D{
					{Key: "account", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "password", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "type", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "active", Value: bson.D{{Key: "bsonType", Value: "long"}}},
					{Key: "detail", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "roleId", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "createAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
					{Key: "updateAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
				}},
			}},
		},
		RoleType: bson.D{
			{Key: "$jsonSchema", Value: bson.D{
				{Key: "bsonType", Value: "object"},
				{Key: "properties", Value: bson.D{
					{Key: "name", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "roles", Value: bson.D{{Key: "bsonType", Value: "array"}}},
					{Key: "createAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
					{Key: "updateAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
				}},
			}},
		},
		WorkFLowType: bson.D{
			{Key: "$jsonSchema", Value: bson.D{
				{Key: "bsonType", Value: "object"},
				{Key: "properties", Value: bson.D{
					{Key: "Channel", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "Topic", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "Gid", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "MessageId", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "Statement", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "Status", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "TaskId", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "CreatedAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
					{Key: "UpdatedAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
				}},
			}},
		},
	}
)

// CollectionType Define the collection types
type CollectionType string

const (
	EventType    CollectionType = "event"
	ManagerType  CollectionType = "manager"
	OptType      CollectionType = "opt"
	RoleType     CollectionType = "role"
	WorkFLowType CollectionType = "workflow"
)

var (
	Collections     []string
	CollectionsOnce sync.Once
)

// Collection represents a MongoDB collection
type Collection string

// Create creates a collection with the specified type
func (t Collection) Create(ctx context.Context, database *mongo.Database, tp CollectionType) error {

	names, err := t.listCollectionNames(ctx, database)
	if err != nil {
		return err
	}
	collectionName := string(t)
	//go1.21+ version
	// Check if the collection already exists
	if b := slices.Contains(names, collectionName); b {
		return nil
	}

	opts, err := t.getCollectionOptions(tp)
	if err != nil {
		return fmt.Errorf("invalid collection type: %w", err)
	}

	if err := database.CreateCollection(ctx, collectionName, opts); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// CreateIndex creates a normal index for the collection
func (t Collection) CreateIndex(ctx context.Context, database *mongo.Database, key string, sort int) error {

	if database == nil {
		return fmt.Errorf("database is nil")
	}

	exists, err := t.checkIndexExists(ctx, database, key)
	if err != nil {
		return err
	}
	// If the index already exists, return
	if exists {
		return nil
	}

	return t.createIndex(ctx, database, key, sort)
}

// CreateTTLIndex creates a TTL index for the collection
func (t Collection) CreateTTLIndex(ctx context.Context, database *mongo.Database, duration time.Duration) error {

	if database == nil {
		return fmt.Errorf("database is nil")
	}

	key := "expireAt"
	exists, err := t.checkIndexExists(ctx, database, key)
	if err != nil {
		return err
	}
	// If the index already exists, return
	if exists {
		return nil
	}

	_, err = database.Collection(string(t)).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: key, Value: 1}},
		Options: options.Index().SetName(fmt.Sprintf("%s_ttl", key)).SetExpireAfterSeconds(cast.ToInt32(duration.Seconds())),
	})

	return err
}

// listCollectionNames returns the list of collection names
func (t Collection) listCollectionNames(ctx context.Context, database *mongo.Database) ([]string, error) {

	if database == nil {
		return nil, fmt.Errorf("database is nil")
	}
	CollectionsOnce.Do(func() {
		if collections, err := database.ListCollectionNames(ctx, bson.M{}); err != nil {
			log.Fatalf("list collection error:%+v \n", err)
		} else {
			Collections = collections
		}
	})
	return Collections, nil
}

// checkIndexExists checks if the specified index exists
func (t Collection) checkIndexExists(ctx context.Context, database *mongo.Database, key string) (bool, error) {

	cursor, err := database.Collection(string(t)).Indexes().List(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		if closeErr := cursor.Close(ctx); closeErr != nil {
			logger.New().Error(closeErr)
		}
	}()

	indexInfo := make(map[string]any, 3)
	for cursor.Next(ctx) {
		if err := cursor.Decode(&indexInfo); err != nil {
			return false, err
		}
		if v, ok := indexInfo["key"]; ok {
			if mv, mok := v.(map[string]any); mok {
				if _, exists := mv[key]; exists {
					return true, nil
				}
			}
		}
	}

	return false, cursor.Err()
}

// createIndex creates an index for the collection
func (t Collection) createIndex(ctx context.Context, database *mongo.Database, key string, sort int) error {
	if database == nil {
		return fmt.Errorf("database is nil")
	}

	_, err := database.Collection(string(t)).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: key, Value: sort}},
		Options: nil,
	})
	return err
}

// getCollectionOptions Get collection creation options
func (t Collection) getCollectionOptions(tp CollectionType) (*options.CreateCollectionOptions, error) {
	if validator, ok := validators[tp]; ok {
		opts := options.CreateCollection().SetValidator(validator)
		if tp == OptType {
			opts.SetValidationLevel("strict")
		}
		return opts, nil
	}

	return nil, fmt.Errorf("unsupported collection type: %v", tp)
}
