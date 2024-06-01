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
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

type (
	scheduleJobI interface {
		enqueue(ctx context.Context, msg *Message) error
		sequentialEnqueue(ctx context.Context, message *Message) error
		sendToStream(ctx context.Context, msg *Message) error
		doConsume(ctx context.Context, max string, channel, topic string) error
	}

	scheduleJob struct {
		broker *RedisBroker
		wg     *sync.WaitGroup

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

	priority := msg.Priority / 1e3
	priority = cast.ToFloat64(msgExecuteTime) + priority
	timeUnit := cast.ToFloat64(msgExecuteTime)

	setKey := MakeZSetKey(t.broker.prefix, msg.Channel, msg.Topic)
	timeUnitKey := MakeTimeUnit(t.broker.prefix, msg.Channel, msg.Topic)

	err = t.broker.client.Watch(ctx, func(tx *redis.Tx) error {

		_, err := tx.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {

			// set value
			if err := pipeliner.ZAdd(ctx, setKey, &redis.Z{Score: priority, Member: bt}).Err(); err != nil {
				return err
			}
			// set time unit
			if err := pipeliner.ZAdd(ctx, timeUnitKey, &redis.Z{Score: timeUnit, Member: timeUnit}).Err(); err != nil {
				return err
			}
			return nil
		})

		return err

	}, setKey, timeUnitKey)

	return err
}

func (t *scheduleJob) doConsume(ctx context.Context, max string, channel, topic string) error {
	zRangeBy := &redis.ZRangeBy{
		Min: defaultScheduleJobConfig.scoreMin,
		Max: max,
	}
	key := MakeZSetKey(t.broker.prefix, channel, topic)

	val := t.broker.client.ZRevRangeByScore(ctx, key, zRangeBy).Val()

	if len(val) <= 0 {
		return nil
	}

	t.doConsumeZset(ctx, val, channel, topic)
	return nil
}

func (t *scheduleJob) doConsumeZset(ctx context.Context, vals []string, channel, topic string) {
	var zsetKey = MakeZSetKey(t.broker.prefix, channel, topic)

	doTask := func(ctx context.Context, vv string) error {

		msg, err := jsonToMessage(vv)
		if err != nil {
			return err
		}

		group := t.scheduleErrGroupPool.Get().(*errgroup.Group)
		group.TryGo(func() error {
			return t.sendToStream(ctx, msg)
		})
		group.TryGo(func() error {
			return t.broker.client.ZRem(ctx, zsetKey, vv).Err()
		})
		if err := group.Wait(); err != nil {
			return err
		}
		t.scheduleErrGroupPool.Put(group)
		return nil
	}
	// begin to execute consumer's datas
	for _, vv := range vals {
		if err := doTask(ctx, vv); err != nil {
			logger.New().With("", err).Error("consumer err")
			continue
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
