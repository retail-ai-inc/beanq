package redisx

import (
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var redisXAddArgsPool = &sync.Pool{New: func() any {
	return &redis.XAddArgs{
		Stream:     "",
		NoMkStream: false,
		MaxLen:     0,
		MinID:      "",
		Approx:     true,
		Limit:      0,
		ID:         "",
		Values:     nil,
	}
}}

func NewZAddArgs(stream, minId, Id string, maxLen, Limit int64, vals any) *redis.XAddArgs {
	args := redisXAddArgsPool.Get().(*redis.XAddArgs)
	args.ID = Id
	args.Stream = stream
	args.MinID = minId
	args.MaxLen = maxLen
	args.Limit = Limit
	args.Values = vals

	defer func() {
		args = &redis.XAddArgs{
			Stream:     "",
			NoMkStream: false,
			MaxLen:     0,
			MinID:      "",
			Approx:     true,
			Limit:      0,
			ID:         "",
			Values:     nil,
		}
		redisXAddArgsPool.Put(args)
	}()
	return args
}

func NewReadGroupArgs(group, consumer string, streams []string, count int64, block time.Duration) *redis.XReadGroupArgs {
	return &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  streams,
		Count:    count,
		Block:    block,
	}
}

var xAutoClaimPool = &sync.Pool{New: func() any {
	return &redis.XAutoClaimArgs{
		Stream:   "",
		Group:    "",
		MinIdle:  0,
		Start:    "",
		Count:    0,
		Consumer: "",
	}
}}

func NewAutoClaimArgs(stream, group string, minIdle time.Duration, start string, count int64, consumer string) *redis.XAutoClaimArgs {
	args := xAutoClaimPool.Get().(*redis.XAutoClaimArgs)
	args.Stream = stream
	args.Group = group
	args.MinIdle = minIdle
	args.Start = start
	args.Count = count
	args.Consumer = consumer
	defer func() {
		args = &redis.XAutoClaimArgs{
			Stream:   "",
			Group:    "",
			MinIdle:  0,
			Start:    "",
			Count:    0,
			Consumer: "",
		}
		xAutoClaimPool.Put(args)
	}()
	return args
}
