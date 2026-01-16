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
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/helper/ui"
	"github.com/retail-ai-inc/beanq/v4/internal/boptions"
	"github.com/spf13/viper"
)

type (
	DebugLog struct {
		Path string `json:"path"`
		On   bool   `json:"on"`
	}
	Health struct {
		Port string `json:"port"`
		Host string `json:"host"`
	}
	Redis struct {
		Host               string        `json:"host"`
		Port               string        `json:"port"`
		Password           string        `json:"password"`
		Prefix             string        `json:"prefix"`
		Database           int           `json:"database"`
		MaxLen             int64         `json:"maxLen"`
		MinIdleConnections int           `json:"minIdleConnections"`
		DialTimeout        time.Duration `json:"dialTimeout"`
		ReadTimeout        time.Duration `json:"readTimeout"`
		WriteTimeout       time.Duration `json:"writeTimeout"`
		PoolTimeout        time.Duration `json:"poolTimeout"`
		MaxRetries         int           `json:"maxRetries"`
		PoolSize           int           `json:"poolSize"`
	}
	Queue struct {
		Topic        string
		DelayChannel string
		DelayTopic   string
		Channel      string
		MaxLen       int64
		Priority     float64
		TimeToRun    time.Duration
	}
	History struct {
		Storage string `json:"storage"`
		On      bool
	}

	UI struct {
		Stmt struct {
			Host     string `json:"host"`
			Port     string `json:"port"`
			User     string `json:"user"`
			Password string `json:"password"`
		}
		GoogleAuth struct {
			ClientId     string
			ClientSecret string
			CallbackUrl  string
		}
		SendGrid struct {
			Key         string
			FromName    string
			FromAddress string
		}
		Root struct {
			UserName string `json:"username"`
			Password string `json:"password"`
		} `json:"root"`
		On        bool          `json:"on"`
		Issuer    string        `json:"issuer"`
		Subject   string        `json:"subject"`
		JwtKey    string        `json:"jwtKey"`
		Port      string        `json:"port"`
		ExpiresAt time.Duration `json:"expiresAt"`
	}
	Collection struct {
		Name  string `json:"name"`
		Shard bool   `json:"shard"`
	}
	Mongo struct {
		Database              string
		UserName              string
		Password              string
		Collections           map[string]Collection
		Host                  string
		Port                  string
		ConnectTimeOut        time.Duration
		MaxConnectionPoolSize uint64
		MaxConnectionLifeTime time.Duration
	}
	BeanqConfig struct {
		Health   Health `json:"health"`
		Broker   string `json:"broker"`
		UI       ui.Ui  `json:"ui"`
		*Mongo   `json:"mongo"`
		DebugLog `json:"debugLog"`
		Queue
		History                  `json:"history"`
		WorkFlow                 `json:"workflow"`
		Redis                    Redis         `json:"redis"`
		DeadLetterIdleTime       time.Duration `json:"deadLetterIdle"`
		DeadLetterTicker         time.Duration `json:"deadLetterTicker"`
		KeepFailedJobsInHistory  time.Duration `json:"keepFailedJobsInHistory"`
		KeepSuccessJobsInHistory time.Duration `json:"keepSuccessJobsInHistory"`
		PublishTimeOut           time.Duration `json:"publishTimeOut"`
		ConsumeTimeOut           time.Duration `json:"consumeTimeOut"`
		MinConsumers             int64         `json:"minConsumers"`
		JobMaxRetries            int           `json:"jobMaxRetries"`
		ConsumerPoolSize         int           `json:"consumerPoolSize"`
	}
)

func (t *BeanqConfig) init() {
	if t.ConsumerPoolSize == 0 {
		t.ConsumerPoolSize = boptions.DefaultOptions.ConsumerPoolSize
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
	if t.MinConsumers == 0 {
		t.MinConsumers = boptions.DefaultOptions.MinConsumers
	}
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
	//nolint:staticcheck,qf1008 //enhance readability
	if t.Mongo.Collections == nil {
		//nolint:staticcheck,qf1008 //enhance readability
		t.Mongo.Collections = map[string]Collection{
			"event":    {Name: "event_logs", Shard: true},
			"workflow": {Name: "workflow_logs", Shard: true},
			"manager":  {Name: "managers", Shard: true},
			"opt":      {Name: "opt_logs", Shard: true},
			"role":     {Name: "roles", Shard: true},
			"tenant":   {Name: "tenants", Shard: true},
		}
	}
	//nolint:staticcheck,qf1008 //enhance readability
	if t.Mongo.ConnectTimeOut == 0 {
		//nolint:staticcheck,qf1008 //enhance readability
		t.Mongo.ConnectTimeOut = 10 * time.Second
	}
	//nolint:staticcheck,qf1008 //enhance readability
	if t.Mongo.MaxConnectionPoolSize == 0 {
		//nolint:staticcheck,qf1008 //enhance readability
		t.Mongo.MaxConnectionPoolSize = 200
	}
	//nolint:staticcheck,qf1008 //enhance readability
	if t.Mongo.MaxConnectionLifeTime == 0 {
		//nolint:staticcheck,qf1008 //enhance readability
		t.Mongo.MaxConnectionLifeTime = 600 * time.Second
	}
}

func (t *BeanqConfig) ToJson() string {

	bt, err := json.Marshal(t)
	if err != nil {
		logger.New().Error(err)
		return ""
	}
	return string(bt)
}

// Default configuration values
const (
	DefaultConfigName = "env"
	DefaultConfigType = "json"
)

var (
	once    sync.Once
	config  BeanqConfig
	initErr error
)

// NewConfig initializes a BeanqConfig from a configuration file using viper.
// It ensures thread-safe initialization and validates inputs.
// Parameters:
//   - configType: Type of configuration file (e.g., "json", "yaml"). Defaults to "json".
//   - configName: Name of the configuration file without extension (e.g., "env"). Defaults to "env".
//   - vp: Optional viper instance for dependency injection (e.g., for testing). If nil, a new instance is created.
//
// Returns a pointer to BeanqConfig and an error if initialization fails.
func NewConfig(configPath string, configType string, configName string) (*BeanqConfig, error) {

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config path %s does not exist", configPath)
	}
	if configType == "" {
		configType = DefaultConfigType
	}
	if configName == "" {
		configName = DefaultConfigName
	}

	once.Do(func() {
		vp := viper.New()
		vp.AddConfigPath(configPath)
		vp.SetConfigType(configType)
		vp.SetConfigName(configName)

		if err := vp.ReadInConfig(); err != nil {
			initErr = fmt.Errorf("failed to read config file: %w", err)
			return
		}

		if err := vp.Unmarshal(&config); err != nil {
			initErr = fmt.Errorf("failed to unmarshal config: %w", err)
			return
		}
	})
	return &config, initErr
}
