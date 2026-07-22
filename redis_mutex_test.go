//go:build integration || ci
// +build integration ci

// WARN: Please use `go test -tags integration ./...` instead of running `go test ./...` if you want to test this file.
package beanq

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestLockContest(t *testing.T) {
	config, err := NewConfig("./", "json", "env.testing")
	assert.NoError(t, err)
	New(config)
	muxClient := NewMuxClient(GetBrokerDriver[redis.UniversalClient]())
	mux := muxClient.NewMutex("test", WithExpiry(time.Second*10))
	err = mux.LockContext(context.Background())
	assert.NoError(t, err)

	bl, err := mux.UnlockContext(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, bl, true)
}
