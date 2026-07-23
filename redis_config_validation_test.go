package beanq

import (
	"strings"
	"testing"
)

func TestRedisClusterRequiresDatabaseZero(t *testing.T) {
	cfg := &BeanqConfig{Broker: "redis", Redis: Redis{IsCluster: true, Host: "127.0.0.1", Port: "6379", Database: 1}}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "database must be 0") {
		t.Fatalf("expected cluster database validation error, got %v", err)
	}
}

func TestStandaloneRedisRejectsMultipleAddresses(t *testing.T) {
	cfg := &BeanqConfig{Broker: "redis", Redis: Redis{Host: "127.0.0.1,127.0.0.2", Port: "6379"}}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "isCluster=true") {
		t.Fatalf("expected explicit cluster validation error, got %v", err)
	}
}
