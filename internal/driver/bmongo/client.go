package bmongo

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/btls"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoLog struct {
	mu                    sync.RWMutex
	client                *mongo.Client
	database              string
	collection            string
	host                  string
	port                  string
	userName              string
	password              string
	sslOn                 bool
	caFile                string
	verifyCertificate     bool
	connectTimeOut        time.Duration
	maxConnectionLifeTime time.Duration
	maxConnectionPoolSize uint64
}

func NewMongoLog(ctx context.Context,
	host, port string,
	connectTimeOut, maxConnectionLifeTime time.Duration,
	maxConnectionPoolSize uint64,
	database, collection, userName, password string,
	sslOn bool, caFile string, verifyCertificate bool, hotReload bool,
) *MongoLog {

	mgoLog := &MongoLog{
		host:                  host,
		port:                  port,
		userName:              userName,
		password:              password,
		database:              database,
		collection:            collection,
		connectTimeOut:        connectTimeOut,
		maxConnectionLifeTime: maxConnectionLifeTime,
		maxConnectionPoolSize: maxConnectionPoolSize,
		sslOn:                 sslOn,
		caFile:                caFile,
		verifyCertificate:     verifyCertificate,
	}

	if err := mgoLog.Reload(ctx); err != nil {
		logger.New().Fatal(err)
	}
	if hotReload && sslOn && caFile != "" {
		if err := btls.WatchCAFile(ctx, "mongo", caFile, mgoLog.Reload); err != nil {
			_ = mgoLog.Close(ctx)
			logger.New().Fatal(err)
		}
	}
	return mgoLog
}

func (t *MongoLog) Migrate(ctx context.Context, data []map[string]any) error {
	datas := make(bson.A, 0, len(data))
	for _, v := range data {
		delete(v, "_id")
		datas = append(datas, bson.M(v))
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	if _, err := t.client.Database(t.database).Collection(t.collection).InsertMany(ctx, datas); err != nil {
		return fmt.Errorf("mongo error:%w", err)
	}
	return nil
}

func (t *MongoLog) Reload(ctx context.Context) error {

	client, err := t.newClient(ctx)
	if err != nil {
		return err
	}

	t.mu.Lock()
	oldClient := t.client
	t.client = client
	t.mu.Unlock()

	if oldClient != nil {
		_ = oldClient.Disconnect(ctx)
	}
	return nil
}

func (t *MongoLog) Close(ctx context.Context) error {

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client == nil {
		return nil
	}

	err := t.client.Disconnect(ctx)
	t.client = nil
	return err
}

func (t *MongoLog) newClient(ctx context.Context) (*mongo.Client, error) {

	opts, err := t.clientOptions()
	if err != nil {
		return nil, err
	}

	mgo, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := mgo.Ping(ctx, nil); err != nil {
		_ = mgo.Disconnect(ctx)
		return nil, err
	}
	return mgo, nil
}

func (t *MongoLog) clientOptions() (*options.ClientOptions, error) {

	port := strings.TrimLeft(t.port, ":")
	port = fmt.Sprintf(":%s", port)
	uri := strings.Join([]string{"mongodb://", t.host, port}, "")

	opts := options.Client().ApplyURI(uri).
		SetConnectTimeout(t.connectTimeOut).
		SetMaxConnIdleTime(t.maxConnectionLifeTime).
		SetMaxPoolSize(t.maxConnectionPoolSize)

	if t.userName != "" && t.password != "" {
		auth := options.Credential{
			AuthSource: t.database,
			Username:   t.userName,
			Password:   t.password,
		}
		opts.SetAuth(auth)
	}

	var tlsConfig *tls.Config
	if t.sslOn {
		var err error
		tlsConfig, err = btls.LoadTLSConfigFromCA(t.caFile, t.verifyCertificate)
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(tlsConfig)
	}
	return opts, nil
}
