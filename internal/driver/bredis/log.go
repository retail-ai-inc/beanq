package bredis

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
)

type migrateStore interface {
	Migrate(ctx context.Context, data []map[string]any) error
}

type Log struct {
	client redis.UniversalClient
	log    migrateStore
	prefix string
}

func NewLog(client redis.UniversalClient, prefix string, log migrateStore) *Log {
	return &Log{
		client: client,
		prefix: prefix,
		log:    log,
	}
}

func (t *Log) Migrate(ctx context.Context, data []map[string]any) error {
	var wait sync.WaitGroup
	for shard := uint64(0); shard < tool.BeanqLogicLogPartitions; shard++ {
		key := tool.MakeLogicShardKey(t.prefix, shard)
		wait.Go(func() {
			t.migrateStream(ctx, key)
		})
	}
	wait.Wait()
	return nil
}

func (t *Log) migrateStream(ctx context.Context, key string) {
	for {
		if ctx.Err() != nil {
			return
		}
		result, err := t.client.XReadGroup(ctx, NewReadGroupArgs(tool.BeanqLogGroup, key, []string{key, ">"}, 1000, 2*time.Second)).Result()
		if err != nil {
			if strings.Contains(err.Error(), "NOGROUP No such") {
				if err := t.client.XGroupCreateMkStream(ctx, key, tool.BeanqLogGroup, "0").Err(); err != nil && !isConsumerGroupExistsError(err) {
					return
				}
				continue
			}
			if errors.Is(err, context.Canceled) {
				return
			}
			if !errors.Is(err, redis.Nil) && !errors.Is(err, redis.ErrClosed) {
				logger.LogRuntimeError(ctx, err)
			}
			continue
		}
		if len(result) == 0 {
			continue
		}
		messages := result[0].Messages
		datas := make([]map[string]any, 0, len(messages))
		ids := make([]string, 0, len(messages))
		for _, message := range messages {
			if message.ID != "" {
				ids = append(ids, message.ID)
				datas = append(datas, message.Values)
			}
		}
		if t.log == nil || len(ids) == 0 {
			continue
		}
		if err := t.log.Migrate(ctx, datas); err != nil {
			logger.LogRuntimeError(ctx, err)
			continue
		}
		if _, err := t.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.XAck(ctx, key, tool.BeanqLogGroup, ids...)
			pipe.XDel(ctx, key, ids...)
			return nil
		}); err != nil {
			logger.LogRuntimeError(ctx, err)
		}
	}
}
