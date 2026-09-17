package beanq

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/spf13/viper"
)

func TestNewConfig(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "env", `{
		"broker":"redis",
        "gracefulShutdownTimeout":"17s",
		"redis":{"host":"localhost","port":"6379"},
		"ui":{"root":{"username":"rai","password":"secret"}}
	}`)

	tests := []struct {
		name          string
		configPath    string
		configType    string
		configName    string
		expectedField string
		expectedErr   string
	}{
		{
			name:          "valid config file",
			configPath:    dir,
			configType:    "json",
			configName:    "env",
			expectedField: "rai",
		},
		{
			name:        "invalid config path",
			configPath:  filepath.Join(dir, "err_filepath"),
			configType:  "json",
			configName:  "env",
			expectedErr: "does not exist",
		},
		{
			name:          "default config type and name",
			configPath:    dir,
			expectedField: "rai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(tt.configPath, tt.configType, tt.configName)
			if tt.expectedErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectedErr)
				} else if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("expected error containing %q, got: %v", tt.expectedErr, err)
				}
			}

			if tt.expectedErr == "" && cfg.GracefulShutdownTimeout != 17*time.Second {
				t.Fatalf("graceful shutdown timeout = %v, want 17s", cfg.GracefulShutdownTimeout)
			}

			if tt.expectedField != "" {
				if cfg == nil {
					t.Fatal("expected non-nil config")
				}
				if tt.expectedField != cfg.UI.Root.UserName {
					t.Errorf("expected field %q, got: %v", tt.expectedField, cfg.UI.Root.UserName)
				}
			}
		})
	}
}

func TestNewConfigLoadsFreshConfigEachCall(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	writeConfig(t, first, "env", `{"broker":"redis","redis":{"host":"first","port":"6379"},"ui":{"root":{"username":"first"}}}`)
	writeConfig(t, second, "env", `{"broker":"redis","redis":{"host":"second","port":"6379"},"ui":{"root":{"username":"second"}}}`)

	cfg1, err := NewConfig(first, "json", "env")
	if err != nil {
		t.Fatalf("first NewConfig error: %v", err)
	}
	cfg2, err := NewConfig(second, "json", "env")
	if err != nil {
		t.Fatalf("second NewConfig error: %v", err)
	}
	if cfg1.Redis.Host != "first" || cfg2.Redis.Host != "second" {
		t.Fatalf("expected fresh configs, got first=%q second=%q", cfg1.Redis.Host, cfg2.Redis.Host)
	}
}

func TestViperUnmarshalRedisSSLCertFile(t *testing.T) {
	vp := viper.New()
	vp.AddConfigPath("./examples/normal")
	vp.SetConfigType("json")
	vp.SetConfigName("env")

	if err := vp.ReadInConfig(); err != nil {
		t.Fatalf("ReadInConfig error: %v", err)
	}

	var cfg BeanqConfig
	if err := vp.Unmarshal(&cfg); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if cfg.Redis.SSL.CAFile == "" {
		t.Fatal("expected redis ssl certFile to unmarshal")
	}
}

func TestBeanqConfigInitCreatesMongoDefaults(t *testing.T) {
	cfg := &BeanqConfig{}
	cfg.init()
	if cfg.Mongo == nil {
		t.Fatal("expected Mongo defaults to be initialized")
	}
	if cfg.Mongo.Collections["event"].Name != "event_logs" {
		t.Fatalf("event collection = %q", cfg.Mongo.Collections["event"].Name)
	}
	if cfg.Mongo.Collections["workflow"].Name != "workflow_records" {
		t.Fatalf("workflow collection = %q", cfg.Mongo.Collections["workflow"].Name)
	}
	if cfg.Channel == "" || cfg.Topic == "" {
		t.Fatal("expected queue defaults to be initialized")
	}
}

func TestBeanqConfigResolveReturnsIndependentValidatedConfig(t *testing.T) {
	source := &BeanqConfig{
		Broker:                  "redis",
		Redis:                   Redis{Host: "localhost", Port: "6379"},
		DefaultPartitions:       7,
		NormalQueuePartitions:   9,
		SequenceQueuePartitions: 11,
		Mongo:                   &Mongo{Collections: map[string]Collection{"event": {Name: "custom-events"}}},
	}

	resolved, err := source.Resolve()
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if source.ConsumerPoolSize != 0 || source.ConsumerReaderPoolSize != 0 {
		t.Fatalf("Resolve modified source defaults: %#v", source)
	}
	resolved.Collections["event"] = Collection{Name: "resolved-events"}
	if source.Collections["event"].Name != "custom-events" {
		t.Fatal("Resolve did not copy nested Mongo collections")
	}

	options := resolved.redisBrokerOptions()
	if options.NormalQueuePartitions != 9 || options.SequenceQueuePartitions != 11 {
		t.Fatalf("resolved partitions = (%d, %d), want (9, 11)", options.NormalQueuePartitions, options.SequenceQueuePartitions)
	}
	if options.ConsumerWorkers == 0 || options.ConsumerReaders == 0 {
		t.Fatalf("resolved runtime pools were not defaulted: %#v", options)
	}
	if options.GracefulShutdownTimeout != 30*time.Second {
		t.Fatalf("graceful shutdown timeout = %v, want 30s", options.GracefulShutdownTimeout)
	}
}

func TestBeanqConfigRejectsNegativeGracefulShutdownTimeout(t *testing.T) {
	cfg := &BeanqConfig{Broker: "redis", Redis: Redis{Host: "localhost"}, GracefulShutdownTimeout: -time.Second}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "gracefulShutdownTimeout") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestBeanqConfigValidateRedisRequired(t *testing.T) {
	cfg := &BeanqConfig{Broker: "redis"}
	cfg.ApplyDefaults()
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !errors.Is(err, berror.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
	if !strings.Contains(err.Error(), "redis.host") {
		t.Fatalf("expected redis.host error, got %v", err)
	}
}

func TestBeanqConfigRedisWait(t *testing.T) {
	cfg := &BeanqConfig{Broker: "redis", Redis: Redis{Host: "localhost", Port: "6379", WaitMode: "wait", WaitReplicas: -1}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "waitReplicas") {
		t.Fatalf("expected waitReplicas validation error, got %v", err)
	}
	cfg.Redis.WaitReplicas = 1
	cfg.Redis.IsCluster = true
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected cluster WAIT config to be valid, got %v", err)
	}
	cfg.Redis.WaitTimeout = -time.Second
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "waitTimeout") {
		t.Fatalf("expected waitTimeout validation error, got %v", err)
	}
	cfg.Redis.WaitTimeout = 0
	cfg.ApplyDefaults()
	if cfg.Redis.WaitTimeout != time.Second {
		t.Fatalf("waitTimeout = %v, want 1s", cfg.Redis.WaitTimeout)
	}
}

func TestBeanqConfigSequenceQueuePartitions(t *testing.T) {
	t.Run("loads configured value", func(t *testing.T) {
		dir := t.TempDir()
		writeConfig(t, dir, "env", `{
			"broker":"redis",
			"redis":{"host":"localhost","port":"6379"},
			"defaultPartitions":7,
			"sequenceQueuePartitions":13
		}`)

		cfg, err := NewConfig(dir, "json", "env")
		if err != nil {
			t.Fatalf("NewConfig error: %v", err)
		}
		if cfg.SequenceQueuePartitions != 13 {
			t.Fatalf("sequenceQueuePartitions = %d, want 13", cfg.SequenceQueuePartitions)
		}
	})

	t.Run("zero remains compatibility sentinel", func(t *testing.T) {
		cfg := &BeanqConfig{DefaultPartitions: 7}
		cfg.ApplyDefaults()
		if cfg.SequenceQueuePartitions != 0 {
			t.Fatalf("sequenceQueuePartitions = %d, want 0 sentinel", cfg.SequenceQueuePartitions)
		}
	})

}

func TestBeanqConfigRuntimeValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*BeanqConfig)
		message string
	}{
		{name: "sequence queue partitions", mutate: func(cfg *BeanqConfig) { cfg.SequenceQueuePartitions = -1 }, message: "sequenceQueuePartitions"},
		{name: "normal queue partitions", mutate: func(cfg *BeanqConfig) { cfg.NormalQueuePartitions = -1 }, message: "normalQueuePartitions"},
		{name: "consumer reader pool", mutate: func(cfg *BeanqConfig) { cfg.ConsumerReaderPoolSize = -1 }, message: "consumerReaderPoolSize"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &BeanqConfig{Broker: "redis", Redis: Redis{Host: "localhost", Port: "6379"}}
			tt.mutate(cfg)
			err := cfg.Validate()
			if err == nil || !errors.Is(err, berror.ErrInvalidConfig) || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("expected ErrInvalidConfig containing %q, got %v", tt.message, err)
			}
		})
	}
}

func TestBeanqConfigConsumerReaderPoolSize(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := &BeanqConfig{}
		cfg.ApplyDefaults()
		if cfg.ConsumerReaderPoolSize != 8 {
			t.Fatalf("consumerReaderPoolSize = %d, want 8", cfg.ConsumerReaderPoolSize)
		}
	})

	t.Run("configured value", func(t *testing.T) {
		dir := t.TempDir()
		writeConfig(t, dir, "env", `{
			"broker":"redis",
			"redis":{"host":"localhost","port":"6379"},
			"consumerReaderPoolSize":12
		}`)
		cfg, err := NewConfig(dir, "json", "env")
		if err != nil {
			t.Fatalf("NewConfig error: %v", err)
		}
		if cfg.ConsumerReaderPoolSize != 12 {
			t.Fatalf("consumerReaderPoolSize = %d, want 12", cfg.ConsumerReaderPoolSize)
		}
	})

}

func TestBeanqConfigValidateMongoWhenHistoryEnabled(t *testing.T) {
	cfg := &BeanqConfig{
		Broker:  "redis",
		Redis:   Redis{Host: "localhost", Port: "6379"},
		History: History{On: true, Storage: "mongo"},
	}
	cfg.ApplyDefaults()
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "mongo.host") {
		t.Fatalf("expected mongo.host error, got %v", err)
	}
}

func TestBeanqConfigSafeJSONMarshalsConfig(t *testing.T) {
	cfg := &BeanqConfig{
		Broker: "redis",
		Redis:  Redis{Host: "localhost", Port: "6379", Password: "redis-secret"},
		Mongo:  &Mongo{Password: "mongo-secret"},
	}
	cfg.UI.JwtKey = "jwt-secret"
	cfg.UI.Root.Password = "root-secret"

	bt, err := cfg.SafeJSON()
	if err != nil {
		t.Fatalf("SafeJSON error: %v", err)
	}
	jsonText := string(bt)
	for _, value := range []string{"redis-secret", "mongo-secret", "jwt-secret", "root-secret"} {
		if !strings.Contains(jsonText, value) {
			t.Fatalf("SafeJSON missing value %q in %s", value, jsonText)
		}
	}
}

func writeConfig(t *testing.T, dir, name, data string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
