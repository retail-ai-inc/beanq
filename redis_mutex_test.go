//go:build ci
// +build ci

// WARN: Please use `go test -tags ci ./...` instead of running `go test ./...` if you want to test this file.
package beanq

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLockContest(t *testing.T) {
	viper.SetConfigFile("env.testing.json")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}
	var config BeanqConfig
	err := viper.Unmarshal(&config)
	assert.NoError(t, err)
	client := New(&config)
	muxClient := NewMuxClient(client.broker.client.(redis.UniversalClient))
	mux := muxClient.NewMutex("test", WithExpiry(time.Second*10))
	err = mux.LockContext(context.Background())
	assert.NoError(t, err)

	bl, err := mux.UnlockContext(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, bl, true)
}
