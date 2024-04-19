package beanq

import (
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

func TestGet(t *testing.T) {

	client := RedisUnique{client: redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        []string{strings.Join([]string{"localhost", "6379"}, ":")},
		Password:     "secret",
		DB:           0,
		MaxRetries:   2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  4 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     30,
		MinIdleConns: 10,
		PoolTimeout:  4 * time.Second,
		PoolFIFO:     true,
	}),
		ticker: time.NewTicker(30 * time.Second),
	}
	client.Add(context.Background(), "zset", "cc")
}
func TestDelete(t *testing.T) {
	client := RedisUnique{client: redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        []string{strings.Join([]string{"localhost", "6379"}, ":")},
		Password:     "secret",
		DB:           0,
		MaxRetries:   2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  4 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     30,
		MinIdleConns: 10,
		PoolTimeout:  4 * time.Second,
		PoolFIFO:     true,
	}),
		ticker: time.NewTicker(5 * time.Second),
	}
	client.Delete(context.Background(), "aa")
}
