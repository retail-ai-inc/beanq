package bredis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/retail-ai-inc/beanq/v3/helper/timex"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/retail-ai-inc/beanq/v3/internal"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
	"strings"
	"sync"
	"time"
)

type (
	Schedule struct {
		preWork PreWork
		base    Base
	}
	// PreWork Check if there are any messages that can be consumed
	PreWork func(ctx context.Context, prefix string, channel, topic string)
)

func NewSchedule(client redis.UniversalClient, prefix string, deadLetterIdle time.Duration) *Schedule {
	work := &Schedule{
		base: Base{
			client:         client,
			IProcessLog:    NewProcessLog(client, prefix),
			subType:        btype.DelaySubscribe,
			prefix:         prefix,
			deadLetterIdle: deadLetterIdle,
			blockDuration:  DefaultBlockDuration,
			errGroup: sync.Pool{New: func() any {
				return new(errgroup.Group)
			}},
		},
	}
	work.preWork = work.PreWork

	return work
}

func (t *Schedule) Enqueue(ctx context.Context, data map[string]any) error {

	bt, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var executeTime time.Time
	var priority float64
	var channel, topic string

	if v, ok := data["executeTime"]; ok {
		executeTime = cast.ToTime(v)
	}
	if v, ok := data["priority"]; ok {
		priority = cast.ToFloat64(v)
	}
	if v, ok := data["channel"]; ok {
		channel = cast.ToString(v)
	}
	if v, ok := data["topic"]; ok {
		topic = cast.ToString(v)
	}

	msgExecuteTime := executeTime.UnixMilli()
	priorityScore := priority / 1e3
	priorityScore = cast.ToFloat64(msgExecuteTime) + priorityScore

	zSetKey := tool.MakeZSetKey(t.base.prefix, channel, topic)

	if err := t.base.client.ZAdd(ctx, zSetKey, &redis.Z{Score: priorityScore, Member: bt}).Err(); err != nil {
		return err
	}

	return err
}

func (t *Schedule) Dequeue(ctx context.Context, channel, topic string, do public.CallBack) {

	go func() {
		t.preWork(ctx, t.base.prefix, channel, topic)
	}()
	go func() {
		t.base.DeadLetter(ctx, channel, topic)
	}()
	t.base.Dequeue(ctx, channel, topic, do)
}

func (t *Schedule) PreWork(ctx context.Context, prefix string, channel, topic string) {

	var (
		zSetKey   = tool.MakeZSetKey(prefix, channel, topic)
		streamKey = tool.MakeStreamKey(t.base.subType, prefix, channel, topic)
	)

	timer := timex.TimerPool.Get(500 * time.Millisecond)
	defer timex.TimerPool.Put(timer)

	for {
		select {
		case <-ctx.Done():
			_ = t.base.client.Close()
			return
		case <-timer.C:

		}
		timer.Reset(1 * time.Second)
		//lock
		lockId := strings.Join([]string{prefix, channel, topic, "lock"}, ":")
		if v := AddLogicLockScript.Run(ctx, t.base.client, []string{lockId}).Val(); v.(int64) == 1 {
			continue
		}

		timeOutKey := cast.ToString(time.Now().UnixMilli() + 1)
		watcher := func(tx *redis.Tx) error {

			vals, err := tx.ZRevRangeByScore(ctx, zSetKey, &redis.ZRangeBy{
				Min:   "0",
				Max:   timeOutKey,
				Count: 100,
			}).Result()
			if err != nil {
				return err
			}
			if len(vals) <= 0 {
				return nil
			}

			if _, err := tx.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {

				delvals := make([]string, len(vals))
				for _, val := range vals {

					data := make(map[string]any, 0)
					if err := tool.JsonDecode(val, &data); err != nil {
						if err := t.base.AddLog(ctx, map[string]any{"moodType": btype.DELAY, "data": val, "addTime": time.Now()}); err != nil {
							logger.New().Error("AddLog Error:", err)
						}
						continue
					}
					delvals = append(delvals, val)
					pipeliner.XAdd(ctx, &redis.XAddArgs{
						Stream: streamKey,
						Approx: false,
						Limit:  0,
						ID:     "*",
						Values: data,
					})
				}
				pipeliner.ZRem(ctx, zSetKey, delvals)
				return nil
			}); err != nil {
				return err
			}
			return nil
		}

		if err := t.base.client.Watch(ctx, watcher, zSetKey, streamKey); err != nil {
			logger.New().Error("Schedule Job Error:", err)
		}
		//release lock
		if err := t.base.client.Del(ctx, lockId).Err(); err != nil {
			logger.New().Error("Schedule Lock Error", err)
		}
	}
}
