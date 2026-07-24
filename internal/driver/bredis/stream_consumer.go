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

type queueBaseOptions struct {
	client                 redis.UniversalClient
	prefix                 string
	consumerPoolSize       int
	consumerReaderPoolSize int
	deadLetterIdle         time.Duration
	captureConfig          *capture.Config
	wait                   replicationWait
}

type queueBase struct {
	client                 redis.UniversalClient
	processLogger          processLogger
	prefix                 string
	consumerPoolSize       int
	consumerReaderPoolSize int
	deadLetterIdle         time.Duration
	captureConfig          *capture.Config
	wait                   replicationWait
}

func newQueueBase(options queueBaseOptions) queueBase {
	if options.consumerReaderPoolSize == 0 {
		options.consumerReaderPoolSize = boptions.DefaultOptions.ConsumerReaderPoolSize
	}
	return queueBase{
		client:                 options.client,
		processLogger:          NewProcessLog(options.client, options.prefix),
		prefix:                 options.prefix,
		consumerPoolSize:       options.consumerPoolSize,
		consumerReaderPoolSize: options.consumerReaderPoolSize,
		deadLetterIdle:         options.deadLetterIdle,
		captureConfig:          options.captureConfig,
		wait:                   options.wait,
	}
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
