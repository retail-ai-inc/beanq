package beanq

import (
	"encoding/json"
	"fmt"
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

func TestTypeConverter(t *testing.T) {
	type Mood string
	var a = struct {
		Mode Mood
	}{
		Mode: "fasdfasdf",
	}
	bs, err := json.Marshal(&a)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(bs))

	var b struct {
		Mode Mood
	}
	err = json.Unmarshal(bs, &b)
	if err != nil {
		panic(err)
	}
	fmt.Println(b.Mode)
}
