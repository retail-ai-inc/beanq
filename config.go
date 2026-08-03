// MIT License

// Copyright The RAI Inc.
// The RAI Authors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package beanq

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/helper/ui"
	"github.com/retail-ai-inc/beanq/v4/internal/boptions"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/bredis"
	"github.com/spf13/viper"
)

type (
	DebugLog struct {
		Path string `json:"path" mapstructure:"path"`
		On   bool   `json:"on" mapstructure:"on"`
	}
	Health struct {
		Port string `json:"port" mapstructure:"port"`
		Host string `json:"host" mapstructure:"host"`
	}
	Redis struct {
		IsCluster          bool          `json:"isCluster" mapstructure:"isCluster"`
		Host               string        `json:"host" mapstructure:"host"`
		Port               string        `json:"port" mapstructure:"port"`
		Username           string        `json:"username" mapstructure:"username"`
		Password           string        `json:"password" mapstructure:"password"`
		Prefix             string        `json:"prefix" mapstructure:"prefix"`
		Database           int           `json:"database" mapstructure:"database"`
		MaxLen             int64         `json:"maxLen" mapstructure:"maxLen"`
		MinIdleConnections int           `json:"minIdleConnections" mapstructure:"minIdleConnections"`
		DialTimeout        time.Duration `json:"dialTimeout" mapstructure:"dialTimeout"`
		ReadTimeout        time.Duration `json:"readTimeout" mapstructure:"readTimeout"`
		WriteTimeout       time.Duration `json:"writeTimeout" mapstructure:"writeTimeout"`
		PoolTimeout        time.Duration `json:"poolTimeout" mapstructure:"poolTimeout"`
		MaxRetries         int           `json:"maxRetries" mapstructure:"maxRetries"`
		PoolSize           int           `json:"poolSize" mapstructure:"poolSize"`
		WaitReplicas       int           `json:"waitReplicas" mapstructure:"waitReplicas"`
		WaitTimeout        time.Duration `json:"waitTimeout" mapstructure:"waitTimeout"`
		SSL                SSL           `json:"ssl" mapstructure:"ssl"`
	}
	SSL struct {
		On        bool   `json:"on" mapstructure:"on"`
		CAFile    string `json:"certFile" mapstructure:"certFile"`
		Verify    bool   `json:"verifyCertificate" mapstructure:"verifyCertificate"`
		HotReload bool   `json:"hotReload" mapstructure:"hotReload"`
	}
	Queue struct {
		Topic        string        `json:"topic" mapstructure:"topic"`
		DelayChannel string        `json:"delayChannel" mapstructure:"delayChannel"`
		DelayTopic   string        `json:"delayTopic" mapstructure:"delayTopic"`
		Channel      string        `json:"channel" mapstructure:"channel"`
		MaxLen       int64         `json:"maxLen" mapstructure:"maxLen"`
		Priority     float64       `json:"priority" mapstructure:"priority"`
		TimeToRun    time.Duration `json:"timeToRun" mapstructure:"timeToRun"`
	}
	History struct {
		Storage string `json:"storage" mapstructure:"storage"`
		On      bool   `json:"on" mapstructure:"on"`
	}

	UI struct {
		Stmt struct {
			Host     string `json:"host" mapstructure:"host"`
			Port     string `json:"port" mapstructure:"port"`
			User     string `json:"user" mapstructure:"user"`
			Password string `json:"password" mapstructure:"password"`
		} `json:"smtp" mapstructure:"smtp"`
		GoogleAuth struct {
			ClientId     string `json:"clientId" mapstructure:"clientId"`
			ClientSecret string `json:"clientSecret" mapstructure:"clientSecret"`
			CallbackUrl  string `json:"callbackUrl" mapstructure:"callbackUrl"`
		} `json:"googleAuth" mapstructure:"googleAuth"`
		SendGrid struct {
			Key         string `json:"key" mapstructure:"key"`
			FromName    string `json:"fromName" mapstructure:"fromName"`
			FromAddress string `json:"fromAddress" mapstructure:"fromAddress"`
		} `json:"sendGrid" mapstructure:"sendGrid"`
		Root struct {
			UserName string `json:"username" mapstructure:"username"`
			Password string `json:"password" mapstructure:"password"`
		} `json:"root" mapstructure:"root"`
		On        bool          `json:"on" mapstructure:"on"`
		Issuer    string        `json:"issuer" mapstructure:"issuer"`
		Subject   string        `json:"subject" mapstructure:"subject"`
		JwtKey    string        `json:"jwtKey" mapstructure:"jwtKey"`
		Port      string        `json:"port" mapstructure:"port"`
		ExpiresAt time.Duration `json:"expiresAt" mapstructure:"expiresAt"`
	}
	Collection struct {
		Name  string `json:"name" mapstructure:"name"`
		Shard bool   `json:"shard" mapstructure:"shard"`
	}
	Mongo struct {
		Database              string                `json:"database" mapstructure:"database"`
		UserName              string                `json:"username" mapstructure:"username"`
		Password              string                `json:"password" mapstructure:"password"`
		Collections           map[string]Collection `json:"collections" mapstructure:"collections"`
		Host                  string                `json:"host" mapstructure:"host"`
		Port                  string                `json:"port" mapstructure:"port"`
		ConnectTimeOut        time.Duration         `json:"connectTimeout" mapstructure:"connectTimeout"`
		MaxConnectionPoolSize uint64                `json:"maxConnectionPoolSize" mapstructure:"maxConnectionPoolSize"`
		MaxConnectionLifeTime time.Duration         `json:"maxConnectionLifeTime" mapstructure:"maxConnectionLifeTime"`
		SSL                   SSL                   `json:"ssl" mapstructure:"ssl"`
	}
	BeanqConfig struct {
		Health                   Health `json:"health" mapstructure:"health"`
		Broker                   string `json:"broker" mapstructure:"broker"`
		UI                       ui.Ui  `json:"ui" mapstructure:"ui"`
		*Mongo                   `json:"mongo" mapstructure:"mongo"`
		DebugLog                 `json:"debugLog" mapstructure:"debugLog"`
		Queue                    `mapstructure:",squash"`
		History                  `json:"history" mapstructure:"history"`
		WorkFlow                 `json:"workflow" mapstructure:"workflow"`
		Redis                    Redis         `json:"redis" mapstructure:"redis"`
		DeadLetterIdleTime       time.Duration `json:"deadLetterIdle" mapstructure:"deadLetterIdle"`
		DeadLetterTicker         time.Duration `json:"deadLetterTicker" mapstructure:"deadLetterTicker"`
		KeepFailedJobsInHistory  time.Duration `json:"keepFailedJobsInHistory" mapstructure:"keepFailedJobsInHistory"`
		KeepSuccessJobsInHistory time.Duration `json:"keepSuccessJobsInHistory" mapstructure:"keepSuccessJobsInHistory"`
		PublishTimeOut           time.Duration `json:"publishTimeOut" mapstructure:"publishTimeOut"`
		ConsumeTimeOut           time.Duration `json:"consumeTimeOut" mapstructure:"consumeTimeOut"`
		GracefulShutdownTimeout  time.Duration `json:"gracefulShutdownTimeout" mapstructure:"gracefulShutdownTimeout"`
		MinConsumers             int64         `json:"minConsumers" mapstructure:"minConsumers"`
		NormalQueuePartitions    int64         `json:"normalQueuePartitions" mapstructure:"normalQueuePartitions"`
		SequenceQueuePartitions  int64         `json:"sequenceQueuePartitions" mapstructure:"sequenceQueuePartitions"`
		JobMaxRetries            int           `json:"jobMaxRetries" mapstructure:"jobMaxRetries"`
		ConsumerPoolSize         int           `json:"consumerPoolSize" mapstructure:"consumerPoolSize"`
		ConsumerReaderPoolSize   int           `json:"consumerReaderPoolSize" mapstructure:"consumerReaderPoolSize"`
	}
	ResolvedConfig struct {
		BeanqConfig
	}
)

// Resolve returns a complete, validated copy without modifying the source config.
func (t *BeanqConfig) Resolve() (ResolvedConfig, error) {
	if t == nil {
		return ResolvedConfig{}, berror.ErrInvalidConfig.WithMessage("config is nil")
	}
	resolved := ResolvedConfig{BeanqConfig: cloneBeanqConfig(*t)}
	resolved.ApplyDefaults()
	if err := resolved.Validate(); err != nil {
		return ResolvedConfig{}, err
	}
	return resolved, nil
}

func cloneBeanqConfig(config BeanqConfig) BeanqConfig {
	if config.Mongo == nil {
		return config
	}
	mongo := *config.Mongo
	if config.Collections != nil {
		mongo.Collections = make(map[string]Collection, len(config.Collections))
		for key, collection := range config.Collections {
			mongo.Collections[key] = collection
		}
	}
	config.Mongo = &mongo
	return config
}

func (t ResolvedConfig) redisBrokerOptions() bredis.BrokerOptions {
	return bredis.BrokerOptions{
		Prefix:                  t.Redis.Prefix,
		MaxLen:                  t.Redis.MaxLen,
		NormalQueuePartitions:   boptions.ResolveNormalQueuePartitions(t.NormalQueuePartitions, t.MinConsumers),
		SequenceQueuePartitions: boptions.ResolveSequenceQueuePartitions(t.SequenceQueuePartitions, t.MinConsumers),
		ConsumerWorkers:         t.ConsumerPoolSize,
		ConsumerReaders:         t.ConsumerReaderPoolSize,
		DeadLetterIdle:          t.DeadLetterIdleTime,
		GracefulShutdownTimeout: t.GracefulShutdownTimeout,
		ReplicationWait: bredis.ReplicationWaitOptions{
			Replicas: t.Redis.WaitReplicas,
			Timeout:  t.Redis.WaitTimeout,
		},
	}
}

func (t ResolvedConfig) redisClientOptions() bredis.RedisClientOptions {
	return bredis.RedisClientOptions{
		IsCluster:          t.Redis.IsCluster,
		Host:               t.Redis.Host,
		Port:               t.Redis.Port,
		Username:           t.Redis.Username,
		Password:           t.Redis.Password,
		Database:           t.Redis.Database,
		MaxRetries:         t.Redis.MaxRetries,
		DialTimeout:        t.Redis.DialTimeout,
		ReadTimeout:        t.Redis.ReadTimeout,
		WriteTimeout:       t.Redis.WriteTimeout,
		PoolTimeout:        t.Redis.PoolTimeout,
		PoolSize:           t.Redis.PoolSize,
		MinIdleConnections: t.Redis.MinIdleConnections,
		TLS: bredis.RedisTLSOptions{
			On:                t.Redis.SSL.On,
			CAFile:            t.Redis.SSL.CAFile,
			VerifyCertificate: t.Redis.SSL.Verify,
			HotReload:         t.Redis.SSL.HotReload,
		},
		WaitReplicas: t.Redis.WaitReplicas,
	}
}

func (t *BeanqConfig) init() {
	t.ApplyDefaults()
}

func (t *BeanqConfig) ApplyDefaults() {
	if t.Mongo == nil {
		t.Mongo = &Mongo{}
	}
	t.applyRuntimeDefaults()
	t.applyRedisDefaults()
	t.applyQueueDefaults()
	t.applyMongoDefaults()
}

func (t *BeanqConfig) applyRedisDefaults() {
	if t.Redis.WaitReplicas > 0 && t.Redis.WaitTimeout == 0 {
		t.Redis.WaitTimeout = time.Second
	}
}

func (t *BeanqConfig) applyRuntimeDefaults() {
	if t.ConsumerPoolSize == 0 {
		t.ConsumerPoolSize = boptions.DefaultOptions.ConsumerPoolSize
	}
	if t.ConsumerReaderPoolSize == 0 {
		t.ConsumerReaderPoolSize = boptions.DefaultOptions.ConsumerReaderPoolSize
	}
	if t.JobMaxRetries < 0 {
		t.JobMaxRetries = boptions.DefaultOptions.JobMaxRetry
	}
	if t.DeadLetterIdleTime == 0 {
		t.DeadLetterIdleTime = boptions.DefaultOptions.DeadLetterIdle
	}
	if t.DeadLetterTicker == 0 {
		t.DeadLetterTicker = boptions.DefaultOptions.DeadLetterTicker
	}
	if t.KeepSuccessJobsInHistory == 0 {
		t.KeepSuccessJobsInHistory = boptions.DefaultOptions.KeepSuccessJobsInHistory
	}
	if t.KeepFailedJobsInHistory == 0 {
		t.KeepFailedJobsInHistory = boptions.DefaultOptions.KeepFailedJobsInHistory
	}
	if t.PublishTimeOut == 0 {
		t.PublishTimeOut = boptions.DefaultOptions.PublishTimeOut
	}
	if t.ConsumeTimeOut == 0 {
		t.ConsumeTimeOut = boptions.DefaultOptions.ConsumeTimeOut
	}
	if t.GracefulShutdownTimeout == 0 {
		t.GracefulShutdownTimeout = boptions.DefaultOptions.GracefulShutdownTimeout
	}
	if t.MinConsumers == 0 {
		t.MinConsumers = boptions.DefaultOptions.MinConsumers
	}
}

func (t *BeanqConfig) applyQueueDefaults() {
	if t.Channel == "" {
		t.Channel = boptions.DefaultOptions.DefaultChannel
	}
	if t.Topic == "" {
		t.Topic = boptions.DefaultOptions.DefaultTopic
	}
	if t.DelayChannel == "" {
		t.DelayChannel = boptions.DefaultOptions.DefaultDelayChannel
	}
	if t.DelayTopic == "" {
		t.DelayTopic = boptions.DefaultOptions.DefaultDelayTopic
	}
	if t.MaxLen == 0 {
		t.MaxLen = boptions.DefaultOptions.DefaultMaxLen
	}
	if t.TimeToRun == 0 {
		t.TimeToRun = boptions.DefaultOptions.TimeToRun
	}
}

func (t *BeanqConfig) applyMongoDefaults() {
	if t.Collections == nil {
		t.Collections = defaultMongoCollections()
	}
	if t.Port == "" {
		t.Port = "27017"
	}
	if t.ConnectTimeOut == 0 {
		t.ConnectTimeOut = 10 * time.Second
	}
	if t.MaxConnectionPoolSize == 0 {
		t.MaxConnectionPoolSize = 200
	}
	if t.MaxConnectionLifeTime == 0 {
		t.MaxConnectionLifeTime = 600 * time.Second
	}
}

func defaultMongoCollections() map[string]Collection {
	return map[string]Collection{
		"config":   {Name: "config", Shard: false},
		"event":    {Name: "event_logs", Shard: true},
		"workflow": {Name: "workflow_records", Shard: true},
		"manager":  {Name: "managers", Shard: false},
		"opt":      {Name: "opt_logs", Shard: true},
		"role":     {Name: "roles", Shard: false},
		"tenant":   {Name: "tenants", Shard: false},
	}
}

func (t *Mongo) collectionNames() map[string]string {
	if t == nil {
		return nil
	}
	collections := make(map[string]string, len(t.Collections))
	for key, collection := range t.Collections {
		collections[key] = collection.Name
	}
	return collections
}

func (t *Mongo) collectionName(key, fallback string) string {
	if t == nil {
		return fallback
	}
	if collection, ok := t.Collections[key]; ok && collection.Name != "" {
		return collection.Name
	}
	return fallback
}

func (t *BeanqConfig) Validate() error {
	if t == nil {
		return berror.ErrInvalidConfig.WithMessage("config is nil")
	}
	if normalizeBrokerName(t.Broker) == "" {
		return berror.ErrInvalidConfig.WithMessage("broker is required")
	}
	if t.SequenceQueuePartitions < 0 {
		return berror.ErrInvalidConfig.WithMessage("sequenceQueuePartitions must not be negative")
	}
	if t.NormalQueuePartitions < 0 {
		return berror.ErrInvalidConfig.WithMessage("normalQueuePartitions must not be negative")
	}
	if t.ConsumerReaderPoolSize < 0 {
		return berror.ErrInvalidConfig.WithMessage("consumerReaderPoolSize must not be negative")
	}
	if t.GracefulShutdownTimeout < 0 {
		return berror.ErrInvalidConfig.WithMessage("gracefulShutdownTimeout must not be negative")
	}
	if err := validateRegisteredBrokerConfig(t); err != nil {
		return err
	}
	if t.requiresMongo() {
		return t.validateMongo()
	}
	return nil
}

func (t *BeanqConfig) validateRedis() error {
	if t.Redis.WaitReplicas < 0 {
		return berror.ErrInvalidConfig.WithMessage("redis.waitReplicas must not be negative")
	}
	if t.Redis.WaitTimeout < 0 {
		return berror.ErrInvalidConfig.WithMessage("redis.waitTimeout must not be negative")
	}
	if strings.TrimSpace(t.Redis.Host) == "" {
		return berror.ErrInvalidConfig.WithMessage("redis.host is required")
	}
	if t.Redis.IsCluster && t.Redis.Database != 0 {
		return berror.ErrInvalidConfig.WithMessage("redis.database must be 0 when redis.isCluster is true")
	}
	if !t.Redis.IsCluster && len(strings.Split(t.Redis.Host, ",")) > 1 {
		return berror.ErrInvalidConfig.WithMessage("multiple redis addresses require redis.isCluster=true")
	}
	if strings.TrimSpace(t.Redis.Port) == "" && !redisHostsIncludePorts(t.Redis.Host) {
		return berror.ErrInvalidConfig.WithMessage("redis.port is required")
	}
	if t.Redis.SSL.On && strings.TrimSpace(t.Redis.SSL.CAFile) == "" {
		return berror.ErrInvalidConfig.WithMessage("redis.ssl.certFile is required when redis ssl is enabled")
	}
	return nil
}

func redisHostsIncludePorts(hosts string) bool {
	for _, host := range strings.Split(hosts, ",") {
		if !strings.Contains(strings.TrimSpace(host), ":") {
			return false
		}
	}
	return true
}

func (t *BeanqConfig) requiresMongo() bool {
	return storageRequiresMongo(t.History.On, t.History.Storage) || storageRequiresMongo(t.WorkFlow.On, t.WorkFlow.Storage)
}

func storageRequiresMongo(on bool, storage string) bool {
	if !on {
		return false
	}
	storage = strings.TrimSpace(strings.ToLower(storage))
	return storage == "" || storage == "mongo"
}

func (t *BeanqConfig) validateMongo() error {
	if t.Mongo == nil {
		return berror.ErrInvalidConfig.WithMessage("mongo config is required")
	}
	if strings.TrimSpace(t.Host) == "" {
		return berror.ErrInvalidConfig.WithMessage("mongo.host is required")
	}
	if strings.TrimSpace(t.Database) == "" {
		return berror.ErrInvalidConfig.WithMessage("mongo.database is required")
	}
	if t.SSL.On && strings.TrimSpace(t.SSL.CAFile) == "" {
		return berror.ErrInvalidConfig.WithMessage("mongo.ssl.certFile is required when mongo ssl is enabled")
	}
	return nil
}

func (t *BeanqConfig) ToJson() string {
	bt, err := t.SafeJSON()
	if err != nil {
		logger.New().Error(err)
		return ""
	}
	return string(bt)
}

func (t *BeanqConfig) SafeJSON() ([]byte, error) {
	if t == nil {
		return json.Marshal((*BeanqConfig)(nil))
	}
	clone := *t
	if t.Mongo != nil {
		mongoClone := *t.Mongo
		clone.Mongo = &mongoClone
	}
	return json.Marshal(clone)
}

// Default configuration values
const (
	DefaultConfigName = "env"
	DefaultConfigType = "json"
)

// NewConfig initializes a BeanqConfig from a configuration file using viper.
// It preserves the historical API while loading a fresh config on every call.
func NewConfig(configPath string, configType string, configName string) (*BeanqConfig, error) {
	return LoadConfig(configPath, configType, configName)
}

func LoadConfig(configPath string, configType string, configName string) (*BeanqConfig, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, berror.ErrInvalidConfig.WithMessage(fmt.Sprintf("config path %s does not exist", configPath)).WithCause(err)
	}
	if configType == "" {
		configType = DefaultConfigType
	}
	if configName == "" {
		configName = DefaultConfigName
	}

	vp := viper.New()
	vp.AddConfigPath(configPath)
	vp.SetConfigType(configType)
	vp.SetConfigName(configName)

	if err := vp.ReadInConfig(); err != nil {
		return nil, berror.ErrInvalidConfig.WithMessage("failed to read config file").WithCause(err)
	}

	var cfg BeanqConfig
	if err := vp.Unmarshal(&cfg); err != nil {
		return nil, berror.ErrInvalidConfig.WithMessage("failed to unmarshal config").WithCause(err)
	}
	resolved, err := cfg.Resolve()
	if err != nil {
		return nil, err
	}
	return &resolved.BeanqConfig, nil
}
