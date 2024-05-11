package beanq

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/retail-ai-inc/beanq/helper/timex"
	"github.com/spf13/cast"
)

type RedisLog struct {
	client redis.UniversalClient
	config *BeanqConfig
	pool   *ants.Pool
}

func NewRedisLog(config *BeanqConfig, pool *ants.Pool, client redis.UniversalClient) *RedisLog {
	return &RedisLog{
		client: client,
		config: config,
		pool:   pool,
	}
}

func (t *RedisLog) Archive(ctx context.Context, result *ConsumerResult) error {

	now := time.Now()
	if result.AddTime == "" {
		result.AddTime = now.Format(timex.DateTime)
	}

	// default ErrorLevel
	key := strings.Join([]string{MakeLogKey(t.config.Redis.Prefix, "fail")}, ":")
	expiration := t.config.KeepFailedJobsInHistory

	// InfoLevel
	if result.Level == InfoLevel {
		key = strings.Join([]string{MakeLogKey(t.config.Redis.Prefix, "success")}, ":")
		expiration = t.config.KeepSuccessJobsInHistory
	}

	result.ExpireTime = time.UnixMilli(now.UnixMilli() + expiration.Milliseconds())

	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("JsonMarshalErr:%s,Stack:%+v", err.Error(), stringx.ByteToString(debug.Stack()))
	}

	return t.client.ZAdd(ctx, key, &redis.Z{
		Score:  float64(result.ExpireTime.UnixMilli()),
		Member: b,
	}).Err()
}

func (t *RedisLog) Obsolete(ctx context.Context) {

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	failKey := MakeLogKey(t.config.Redis.Prefix, "fail")
	successKey := MakeLogKey(t.config.Redis.Prefix, "success")

	for {
		// check state
		select {
		case <-ctx.Done():
			logger.New().Info("-------Redis Obsolete Stop-----------")
			return
		case <-ticker.C:
		}

		// delete fail logs
		if err := t.pool.Submit(func() {
			t.client.ZRemRangeByScore(ctx, failKey, "0", cast.ToString(time.Now().UnixMilli()))
		}); err != nil {
			logger.New().Error(err)
		}
		// delete success logs
		if err := t.pool.Submit(func() {
			t.client.ZRemRangeByScore(ctx, successKey, "0", cast.ToString(time.Now().UnixMilli()))
		}); err != nil {
			logger.New().Error(err)
		}
	}
}
