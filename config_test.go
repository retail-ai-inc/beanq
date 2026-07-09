package beanq

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestNewConfig(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "env", `{
		"broker":"redis",
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

func TestBeanqConfigValidateRedisRequired(t *testing.T) {
	cfg := &BeanqConfig{Broker: "redis"}
	cfg.ApplyDefaults()
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
	if !strings.Contains(err.Error(), "redis.host") {
		t.Fatalf("expected redis.host error, got %v", err)
	}
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
