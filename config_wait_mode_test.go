package beanq

import (
	"strings"
	"testing"
	"time"
)

func TestBeanqConfigRedisWaitMode(t *testing.T) {
	t.Run("rejects unsupported mode", func(t *testing.T) {
		cfg := &BeanqConfig{Broker: "redis", Redis: Redis{
			Host: "localhost", Port: "6379", WaitMode: "unknown",
		}}
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "waitMode") {
			t.Fatalf("expected waitMode validation error, got %v", err)
		}
	})

	t.Run("requires WAIT replicas", func(t *testing.T) {
		cfg := &BeanqConfig{Broker: "redis", Redis: Redis{
			Host: "localhost", Port: "6379", WaitMode: "wait",
		}}
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "waitReplicas") {
			t.Fatalf("expected waitReplicas validation error, got %v", err)
		}
	})

	t.Run("maps WAITAOF settings", func(t *testing.T) {
		cfg := &BeanqConfig{Broker: "redis", Redis: Redis{
			Host: "localhost", Port: "6379", WaitMode: "waitaof",
			WaitAOFLocal: 1, WaitReplicas: 2,
		}}
		cfg.ApplyDefaults()
		if err := cfg.Validate(); err != nil {
			t.Fatal(err)
		}
		if cfg.Redis.WaitTimeout != time.Second {
			t.Fatalf("waitTimeout = %v, want 1s", cfg.Redis.WaitTimeout)
		}
		resolved, err := cfg.Resolve()
		if err != nil {
			t.Fatal(err)
		}
		brokerOptions := resolved.redisBrokerOptions().ReplicationWait
		clientOptions := resolved.redisClientOptions()
		if brokerOptions.Mode != "waitaof" || brokerOptions.AOFLocal != 1 || brokerOptions.Replicas != 2 {
			t.Fatalf("unexpected broker WAITAOF options: %#v", brokerOptions)
		}
		if clientOptions.WaitMode != "waitaof" || clientOptions.WaitAOFLocal != 1 || clientOptions.WaitReplicas != 2 {
			t.Fatalf("unexpected client WAITAOF options: %#v", clientOptions)
		}
	})
}
