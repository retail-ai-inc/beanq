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

	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"github.com/spf13/cast"
)

type (
	scheduleJobI interface {
		start(ctx context.Context, consumer *ConsumerHandler) error
		enqueue(ctx context.Context, msg *Message, option Option) error
		shutDown()
		sendToStream(ctx context.Context, msg *Message) error
	}
	scheduleJob struct {
		client     redis.UniversalClient
		wg         *sync.WaitGroup
		pool       *ants.Pool
		stop, done chan struct{}
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

func newScheduleJob(pool *ants.Pool, client redis.UniversalClient) *scheduleJob {
	return &scheduleJob{client: client, wg: &sync.WaitGroup{}, pool: pool, stop: make(chan struct{}), done: make(chan struct{})}
}

func (t *scheduleJob) start(ctx context.Context, consumer *ConsumerHandler) error {

	if err := t.pool.Submit(func() {
		t.consume(ctx, consumer)
	}); err != nil {
		return err
	}
	return nil
}

func (t *scheduleJob) enqueue(ctx context.Context, msg *Message, opt Option) error {

	bt, err := json.Marshal(msg.Values)
	if err != nil {
		return err
	}
	msgExecuteTime := msg.ExecuteTime().UnixMilli()

	priority := opt.Priority / 1e3
	priority = cast.ToFloat64(msgExecuteTime) + priority
	timeUnit := cast.ToFloat64(msgExecuteTime)

	setKey := MakeZSetKey(Config.Redis.Prefix, opt.Channel, opt.Topic)
	timeUnitKey := MakeTimeUnit(Config.Redis.Prefix, opt.Channel, opt.Topic)

	err = t.client.Watch(ctx, func(tx *redis.Tx) error {

		_, err := tx.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {

			// set value
			if err := pipeliner.ZAdd(ctx, setKey, redis.Z{Score: priority, Member: bt}).Err(); err != nil {
				return err
			}
			// set time unit
			if err := pipeliner.ZAdd(ctx, timeUnitKey, redis.Z{Score: timeUnit, Member: timeUnit}).Err(); err != nil {
				return err
			}
			return nil
		})

		return err

	}, setKey, timeUnitKey)

	return err
}

func (t *scheduleJob) consume(ctx context.Context, consumer *ConsumerHandler) {

	ticker := time.NewTicker(defaultScheduleJobConfig.consumeTicker)
	defer ticker.Stop()

	var (
		now      time.Time
		timeUnit = MakeTimeUnit(Config.Redis.Prefix, consumer.Channel, consumer.Topic)
	)
	for {
		select {
		case <-ctx.Done():
			t.pool.Release()
			return
		case <-t.done:
			t.pool.Release()
			return
		case <-ticker.C:

			now = time.Now()

			max := cast.ToString(now.UnixMilli() + 10)

			cmd := t.client.ZRangeByScore(ctx, timeUnit, &redis.ZRangeBy{
				Min:    "0",
				Max:    max,
				Offset: 0,
				Count:  1,
			})

			if err := cmd.Err(); err != nil {
				logger.New().With("", err).Error("consume err")
			}
			val := cmd.Val()
			if len(val) <= 0 {
				continue
			}

			if err := t.client.ZRem(ctx, timeUnit, val[0]).Err(); err != nil {
				logger.New().With("", err).Error("zrem err")
			}

			if err := t.doConsume(ctx, max, consumer); err != nil {
				logger.New().With("", err).Error("consume err")
				// continue
			}

		}
	}
}

func (t *scheduleJob) doConsume(ctx context.Context, max string, consumer *ConsumerHandler) error {

	zRangeBy := &redis.ZRangeBy{
		Min: defaultScheduleJobConfig.scoreMin,
		Max: max,
	}
	key := MakeZSetKey(Config.Redis.Prefix, consumer.Channel, consumer.Topic)

	cmd := t.client.ZRevRangeByScore(ctx, key, zRangeBy)
	if err := cmd.Err(); err != nil && err != redis.Nil {
		return err
	}

	val := cmd.Val()
	if len(val) <= 0 {
		return nil
	}

	t.doConsumeZset(ctx, val, consumer)
	return nil
}

func (t *scheduleJob) doConsumeZset(ctx context.Context, vals []string, consumer *ConsumerHandler) {

	var zsetKey = MakeZSetKey(Config.Redis.Prefix, consumer.Channel, consumer.Topic)

	doTask := func(ctx context.Context, vv string, consumer *ConsumerHandler) error {
		msg, err := jsonToMessage(vv)
		if err != nil {
			return err
		}

		if err := t.sendToStream(ctx, msg); err != nil {
			return err
		}

		// Delete data from `zset`
		if err := t.client.ZRem(ctx, zsetKey, vv).Err(); err != nil {
			return err
		}
		return nil
	}
	// begin to execute consumer's datas
	for _, vv := range vals {
		if err := doTask(ctx, vv, consumer); err != nil {
			logger.New().With("", err).Error("consumer err")
			continue
		}
	}
}

func (t *scheduleJob) sendToStream(ctx context.Context, msg *Message) error {

	xAddArgs := redisx.NewZAddArgs(MakeStreamKey(Config.Redis.Prefix, msg.Channel(), msg.Topic()), "", "*", Config.Redis.MaxLen, 0, map[string]any(msg.Values))
	return t.client.XAdd(ctx, xAddArgs).Err()
}

func (t *scheduleJob) shutDown() {
	t.done <- struct{}{}
}
