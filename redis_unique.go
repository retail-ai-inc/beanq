package beanq

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/spf13/cast"
)

type RedisUnique struct {
	client redis.UniversalClient
	ticker *time.Ticker
}

func (t *RedisUnique) Add(ctx context.Context, key, member string) (bool, error) {

	incr := 0.000
	b := false
	err := t.client.ZRank(ctx, key, member).Err()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			now := time.Now().Unix()
			incr = float64(now) + 0.001
		}
		return b, t.client.ZIncrBy(ctx, key, incr, member).Err()
	}
	incr = 0.001
	b = true
	return b, t.client.ZIncrBy(ctx, key, incr, member).Err()

}

func (t *RedisUnique) Delete(ctx context.Context, key string) {

	defer func() {
		t.ticker.Stop()
	}()
	for {
		select {
		case <-ctx.Done():
			logger.New().Info("Obsolete Task Stop")
			return
		case <-t.ticker.C:

			cmd := t.client.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
				Min:    "-inf",
				Max:    "+inf",
				Offset: 0,
				Count:  100,
			})
			val := cmd.Val()
			if len(val) <= 0 {
				continue
			}

			for _, v := range val {

				floor := math.Floor(v.Score)
				frac := v.Score - floor
				expTime := cast.ToTime(cast.ToInt(floor))

				if time.Since(expTime).Seconds() >= 3600*2 {
					t.client.ZRem(ctx, key, v.Member)
					continue
				}
				if time.Since(expTime).Seconds() >= 60*30 && frac*1000 <= 2 {
					t.client.ZRem(ctx, key, v.Member)
					continue
				}
			}
		}
	}
}
