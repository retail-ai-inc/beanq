package bredis

import (
	"time"

	"github.com/go-redis/redis/v8"
)

func NewZAddArgs(stream, minId, Id string, maxLen, Limit int64, vals any) *redis.XAddArgs {

	return &redis.XAddArgs{
		Stream:     stream,
		NoMkStream: false,
		MaxLen:     maxLen,
		MinID:      minId,
		Approx:     false,
		Limit:      Limit,
		ID:         Id,
		Values:     vals,
	}
}

func NewReadGroupArgs(group, consumer string, streams []string, count int64, block time.Duration) *redis.XReadGroupArgs {
	args := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  streams,
		Count:    count,
	}
	if block > 0 {
		args.Block = block
	}
	return args
}
