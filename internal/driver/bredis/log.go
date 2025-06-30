package bredis

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/helper/timex"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	public "github.com/retail-ai-inc/beanq/v4/internal"
)

type Log struct {
	client redis.UniversalClient
	log    public.IMigrateLog
	prefix string
}

func NewLog(client redis.UniversalClient, prefix string, log public.IMigrateLog) *Log {
	return &Log{
		client: client,
		prefix: prefix,
		log:    log,
	}
}

func (t *Log) Migrate(ctx context.Context, data []map[string]any) error {

	timer := timex.TimerPool.Get(5 * time.Second)
	defer timex.TimerPool.Put(timer)

	key := tool.MakeLogicKey(t.prefix)

	for {
		// check state
		select {
		case <-ctx.Done():
			_ = t.client.Close()
			return nil
		case <-timer.C:
		}
		timer.Reset(5 * time.Second)

		result, err := t.client.XReadGroup(ctx, NewReadGroupArgs(tool.BeanqLogGroup, key, []string{key, ">"}, 200, 20*time.Second)).Result()
		if err != nil {
			if strings.Contains(err.Error(), "NOGROUP No such") {
				if err := t.client.XGroupCreateMkStream(ctx, key, tool.BeanqLogGroup, "0").Err(); err != nil {
					//t.captureException(ctx, err)
					return nil
				}
				continue
			}
			if errors.Is(err, context.Canceled) {
				logger.New().Info("Redis Obsolete Stop")
				return nil
			}
			if !errors.Is(err, redis.Nil) && !errors.Is(err, redis.ErrClosed) {
				logger.New().Error(err)
			}
			continue
		}

		if len(result) <= 0 {
			continue
		}

		messages := result[0].Messages
		datas := make([]map[string]any, 0, len(messages))
		ids := make([]string, 0, len(messages))

		for _, v := range messages {
			if v.ID != "" {
				ids = append(ids, v.ID)
				datas = append(datas, v.Values)
			}
		}
		if t.log != nil {
			if err := t.log.Migrate(ctx, datas); err != nil {
				logger.New().Error(err)
				continue
			}
			if _, err := t.client.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
				pipeliner.XAck(ctx, key, tool.BeanqLogGroup, ids...)
				pipeliner.XDel(ctx, key, ids...)
				return nil
			}); err != nil {
				logger.New().Error(err)
			}
		}
	}
}
