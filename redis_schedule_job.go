// MIT License

// Copyright The RAI Inc.
// The RAI Authors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package beanq

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/retail-ai-inc/beanq/v3/helper/redisx"
	"github.com/retail-ai-inc/beanq/v3/helper/timex"
	"github.com/spf13/cast"
)

type (
	scheduleJobI interface {
		enqueue(ctx context.Context, msg *Message) error
		sequentialEnqueue(ctx context.Context, message *Message) error
		sendToStream(ctx context.Context, msg *Message) error
		run(ctx context.Context, channel, topic string, closeCh chan struct{}) error
	}

	scheduleJob struct {
		broker               *RedisBroker
		wg                   *sync.WaitGroup
		scheduleErrGroupPool *sync.Pool
	}
)

var (
	_ scheduleJobI = (*scheduleJob)(nil)
	// schedule job config
	defaultScheduleJobConfig = struct {
		// zset attribute score,default 0-10
		scoreMin, scoreMax string
		// zset data limit
		offset, count int64
		// delayJob and consumer executeTime
		delayJobTicker, consumeTicker time.Duration
	}{
		scoreMin:       "-inf",
		scoreMax:       "10",
		offset:         0,
		count:          -1,
		delayJobTicker: 10 * time.Second,
		consumeTicker:  1 * time.Second,
	}
)

func (t *scheduleJob) enqueue(ctx context.Context, msg *Message) error {
	bt, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	msgExecuteTime := msg.ExecuteTime.UnixMilli()

	priorityScore := msg.Priority / 1e3
	priorityScore = cast.ToFloat64(msgExecuteTime) + priorityScore

	zSetKey := MakeZSetKey(t.broker.prefix, msg.Channel, msg.Topic)

	if err := t.broker.client.ZAdd(ctx, zSetKey, &redis.Z{Score: priorityScore, Member: bt}).Err(); err != nil {
		return err
	}

	return err
}

func (t *scheduleJob) run(ctx context.Context, channel, topic string, closeCh chan struct{}) error {

	var (
		zSetKey   = MakeZSetKey(t.broker.prefix, channel, topic)
		streamKey = MakeStreamKey(normalSubscribe, t.broker.prefix, channel, topic)
	)

	timer := timex.TimerPool.Get(500 * time.Millisecond)
	defer timex.TimerPool.Put(timer)

	for {
		select {
		case <-closeCh:
			return nil
		case <-ctx.Done():
			logger.New().Info("Channel:[", channel, "]Topic:[", topic, "],Schedule Task Stop")
			return nil
		case <-timer.C:

		}
		timer.Reset(1 * time.Second)
		//lock
		lockId := strings.Join([]string{t.broker.prefix, channel, topic, "lock"}, ":")
		if v := redisx.AddLogicLockScript.Run(ctx, t.broker.client, []string{lockId}).Val(); v.(int64) == 1 {
			continue
		}

		timeOutKey := cast.ToString(time.Now().UnixMilli() + 1)

		err := t.broker.client.Watch(ctx, func(tx *redis.Tx) error {
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
			for _, val := range vals {
				data, err := jsonToMap(val)
				if err != nil {
					return err
				}
				if _, err := tx.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
					pipeliner.XAdd(ctx, &redis.XAddArgs{
						Stream: streamKey,
						Approx: false,
						Limit:  0,
						ID:     "*",
						Values: data,
					})
					pipeliner.ZRem(ctx, zSetKey, val)
					return nil
				}); err != nil {
					logger.New().Error("Schedule Pipeline Error:", err)
					continue
				}
			}
			return nil
		}, zSetKey, streamKey)
		if err != nil {
			logger.New().Error("Schedule Job Error:", err)
		}
		//release lock
		if err := t.broker.client.Del(ctx, lockId).Err(); err != nil {
			logger.New().Error("Schedule Lock Error", err)
		}
	}
}

func (t *scheduleJob) sendToStream(ctx context.Context, msg *Message) error {
	subType := normalSubscribe
	if msg.MoodType == SEQUENTIAL {
		subType = sequentialSubscribe
	}
	xAddArgs := redisx.NewZAddArgs(MakeStreamKey(subType, t.broker.prefix, msg.Channel, msg.Topic), "", "*", t.broker.maxLen, 0, msg.ToMap())
	return t.broker.client.XAdd(ctx, xAddArgs).Err()
}

func (t *scheduleJob) sequentialEnqueue(ctx context.Context, msg *Message) error {
	args := redisx.NewZAddArgs(MakeStreamKey(sequentialSubscribe, t.broker.prefix, msg.Channel, msg.Topic), "", "*", msg.MaxLen, 0, msg.ToMap())
	return t.broker.client.XAdd(ctx, args).Err()
}
