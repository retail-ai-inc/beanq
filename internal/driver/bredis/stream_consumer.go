package bredis

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/internal/boptions"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
)

type BlockDuration func() time.Duration

type queueRuntimeOptions struct {
	workers int
	readers int
}

type queueOptions struct {
	client                  redis.UniversalClient
	prefix                  string
	maxLen                  int64
	partitions              int64
	runtime                 queueRuntimeOptions
	deadLetterIdle          time.Duration
	gracefulShutdownTimeout time.Duration
	captureConfig           *capture.Config
	wait                    replicationWait
}

type queueBase struct {
	client                  redis.UniversalClient
	processLogger           processLogger
	prefix                  string
	consumerPoolSize        int
	consumerReaderPoolSize  int
	deadLetterIdle          time.Duration
	gracefulShutdownTimeout time.Duration
	captureConfig           *capture.Config
	wait                    replicationWait
}

func newQueueBase(options queueOptions) queueBase {
	if options.runtime.readers == 0 {
		options.runtime.readers = boptions.DefaultOptions.ConsumerReaderPoolSize
	}
	return queueBase{
		client:                  options.client,
		processLogger:           NewProcessLog(options.client, options.prefix),
		prefix:                  options.prefix,
		consumerPoolSize:        options.runtime.workers,
		consumerReaderPoolSize:  options.runtime.readers,
		deadLetterIdle:          options.deadLetterIdle,
		gracefulShutdownTimeout: options.gracefulShutdownTimeout,
		captureConfig:           options.captureConfig,
		wait:                    options.wait,
	}
}

func (b *queueBase) GracefulShutdownTimeout() time.Duration {
	if b.gracefulShutdownTimeout > 0 {
		return b.gracefulShutdownTimeout
	}
	return boptions.DefaultOptions.GracefulShutdownTimeout
}

func (b *queueBase) addLog(ctx context.Context, data map[string]any) error {
	return b.processLogger.AddLog(ctx, data)
}

//nolint:gosec
var DefaultBlockDuration BlockDuration = func() time.Duration {
	return time.Duration(rand.Int63n(9)+1) * time.Second
}

func isConsumerGroupExistsError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "BUSYGROUP Consumer Group name already exists")
}
