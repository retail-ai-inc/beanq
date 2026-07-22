package bmongo

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq/v4/internal/driver/btls"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const mongoDisconnectTimeout = 10 * time.Second

var ErrMongoStoreClosed = errors.New("mongo store is closed")

type Config struct {
	Host              string
	Port              string
	Database          string
	Collection        string
	Username          string
	Password          string
	ConnectTimeout    time.Duration
	MaxConnIdleTime   time.Duration
	MaxPoolSize       uint64
	TLSOn             bool
	CAFile            string
	VerifyCertificate bool
	HotReload         bool
}

type MongoStore struct {
	mu                    sync.RWMutex
	reloadMu              sync.Mutex
	client                *mongo.Client
	closed                bool
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
	maxConnIdleTime       time.Duration
	maxConnectionPoolSize uint64
}

func NewMongoStore(ctx context.Context, config Config) (*MongoStore, error) {
	store := &MongoStore{
		host:                  config.Host,
		port:                  config.Port,
		userName:              config.Username,
		password:              config.Password,
		database:              config.Database,
		collection:            config.Collection,
		connectTimeOut:        config.ConnectTimeout,
		maxConnIdleTime:       config.MaxConnIdleTime,
		maxConnectionPoolSize: config.MaxPoolSize,
		sslOn:                 config.TLSOn,
		caFile:                config.CAFile,
		verifyCertificate:     config.VerifyCertificate,
	}

	if err := store.Reload(ctx); err != nil {
		return nil, fmt.Errorf("initialize mongo store: %w", err)
	}
	if config.HotReload && config.TLSOn && config.CAFile != "" {
		if err := btls.WatchCAFile(ctx, "mongo", config.CAFile, store.Reload); err != nil {
			_ = store.Close(context.Background())
			return nil, fmt.Errorf("watch mongo CA file: %w", err)
		}
	}
	return store, nil
}

func (t *MongoStore) InsertMany(ctx context.Context, data []map[string]any) error {
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

func (t *MongoStore) Migrate(ctx context.Context, data []map[string]any) error {
	return t.InsertMany(ctx, data)
}

func (t *MongoStore) Reload(ctx context.Context) error {
	t.reloadMu.Lock()
	defer t.reloadMu.Unlock()

	client, err := t.newClient(ctx)
	if err != nil {
		return fmt.Errorf("create replacement mongo client: %w", err)
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		_ = disconnectMongoClient(client, context.Background())
		return ErrMongoStoreClosed
	}
	oldClient := t.client
	t.client = client
	t.mu.Unlock()

	if oldClient != nil {
		if err := disconnectMongoClient(oldClient, context.Background()); err != nil {
			return fmt.Errorf("disconnect previous mongo client: %w", err)
		}
	}
	return nil
}

func (t *MongoStore) Close(ctx context.Context) error {
	t.reloadMu.Lock()
	defer t.reloadMu.Unlock()
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	client := t.client
	t.client = nil
	t.mu.Unlock()
	if client == nil {
		return nil
	}
	if err := disconnectMongoClient(client, ctx); err != nil {
		return fmt.Errorf("disconnect mongo client: %w", err)
	}
	return nil
}

func (t *MongoStore) newClient(ctx context.Context) (*mongo.Client, error) {

	opts, err := t.clientOptions()
	if err != nil {
		return nil, err
	}

	connectTimeout := t.connectTimeOut
	if connectTimeout <= 0 {
		connectTimeout = 10 * time.Second
	}
	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	mgo, err := mongo.Connect(connectCtx, opts)
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}
	if err := mgo.Ping(connectCtx, nil); err != nil {
		_ = disconnectMongoClient(mgo, context.Background())
		return nil, fmt.Errorf("ping mongo: %w", err)
	}
	return mgo, nil
}

func (t *MongoStore) clientOptions() (*options.ClientOptions, error) {

	port := strings.TrimPrefix(strings.TrimSpace(t.port), ":")
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("invalid mongo port %q", t.port)
	}
	host := strings.TrimSpace(t.host)
	if host == "" {
		return nil, errors.New("mongo host is required")
	}
	address := net.JoinHostPort(host, strconv.Itoa(portNumber))

	opts := options.Client().SetHosts([]string{address}).
		SetConnectTimeout(t.connectTimeOut).
		SetMaxConnIdleTime(t.maxConnIdleTime).
		SetMaxPoolSize(t.maxConnectionPoolSize)

	if t.userName != "" {
		authSource := t.database
		if authSource == "" {
			authSource = "admin"
		}
		auth := options.Credential{
			AuthSource: authSource,
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

func disconnectMongoClient(client *mongo.Client, parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, mongoDisconnectTimeout)
	defer cancel()
	return client.Disconnect(ctx)
}
