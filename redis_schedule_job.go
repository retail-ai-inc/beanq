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
	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"github.com/spf13/cast"
)

type (
	scheduleJobI interface {
		start(ctx context.Context, consumer *ConsumerHandler) error
		enqueue(ctx context.Context, msg *Message, option Option) error
		sequentialEnqueue(ctx context.Context, message *Message, option Option) error
		shutDown()
		sendToStream(ctx context.Context, msg *Message) error
	}
	scheduleJob struct {
		client                    redis.UniversalClient
		wg                        *sync.WaitGroup
		pool                      *ants.Pool
		stop, done, seqDone       chan struct{}
		scheduleTicker, seqTicker *time.Ticker

		prefix string
		maxLen int64
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
	prefix := Config.Load().(BeanqConfig).Redis.Prefix
	if prefix == "" {
		prefix = DefaultOptions.Prefix
	}
	maxLen := Config.Load().(BeanqConfig).Redis.MaxLen
	if maxLen <= 0 {
		maxLen = DefaultOptions.DefaultMaxLen
	}
	return &scheduleJob{
		client:         client,
		wg:             &sync.WaitGroup{},
		pool:           pool,
		stop:           make(chan struct{}),
		done:           make(chan struct{}),
		seqDone:        make(chan struct{}),
		scheduleTicker: time.NewTicker(defaultScheduleJobConfig.consumeTicker),
		seqTicker:      time.NewTicker(10 * time.Second),
		prefix:         prefix,
		maxLen:         maxLen,
	}

}

func (t *scheduleJob) start(ctx context.Context, consumer *ConsumerHandler) error {

	if err := t.pool.Submit(func() {
		t.consume(ctx, consumer)
	}); err != nil {
		return err
	}

	if err := t.pool.Submit(func() {
		t.consumeSeq(ctx, consumer)
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

	setKey := MakeZSetKey(t.prefix, opt.Channel, opt.Topic)
	timeUnitKey := MakeTimeUnit(t.prefix, opt.Channel, opt.Topic)

	err = t.client.Watch(ctx, func(tx *redis.Tx) error {

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

func (t *scheduleJob) consume(ctx context.Context, consumer *ConsumerHandler) {

	// timeWheel To be implemented

	defer t.scheduleTicker.Stop()

	var (
		now      time.Time
		timeUnit = MakeTimeUnit(t.prefix, consumer.Channel, consumer.Topic)
	)
	for {
		select {
		case <-ctx.Done():
			t.pool.Release()
			return
		case <-t.done:
			t.pool.Release()
			return
		case <-t.scheduleTicker.C:
		}

		now = time.Now()

		max := cast.ToString(now.UnixMilli() + 1)

		val, err := t.client.ZRangeByScore(ctx, timeUnit, &redis.ZRangeBy{
			Min:    "0",
			Max:    max,
			Offset: 0,
			Count:  1,
		}).Result()

		if err != nil {
			logger.New().With("", err).Error("consume err")
		}

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

func (t *scheduleJob) doConsume(ctx context.Context, max string, consumer *ConsumerHandler) error {

	zRangeBy := &redis.ZRangeBy{
		Min: defaultScheduleJobConfig.scoreMin,
		Max: max,
	}
	key := MakeZSetKey(t.prefix, consumer.Channel, consumer.Topic)

	val := t.client.ZRevRangeByScore(ctx, key, zRangeBy).Val()

	if len(val) <= 0 {
		return nil
	}

	t.doConsumeZset(ctx, val, consumer)
	return nil
}

func (t *scheduleJob) doConsumeZset(ctx context.Context, vals []string, consumer *ConsumerHandler) {

	var zsetKey = MakeZSetKey(t.prefix, consumer.Channel, consumer.Topic)

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

	xAddArgs := redisx.NewZAddArgs(MakeStreamKey(t.prefix, msg.Channel(), msg.Topic()), "", "*", t.maxLen, 0, msg.Values)
	return t.client.XAdd(ctx, xAddArgs).Err()
}

// can order consume
func (t *scheduleJob) sequentialEnqueue(ctx context.Context, message *Message, opt Option) error {

	bt, err := json.Marshal(message)
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()

	key := MakeListKey(t.prefix, opt.Channel, opt.Topic)

	valKey := strings.Join([]string{opt.OrderKey, cast.ToString(now)}, "_")
	value := strings.Join([]string{valKey, string(bt)}, ":")

	if err := t.client.LPush(ctx, key, value).Err(); err != nil {
		return err
	}
	return nil
}

// Autonomous sorting
func (t *scheduleJob) consumeSeq(ctx context.Context, handler *ConsumerHandler) {

	defer t.seqTicker.Stop()
	// sort orderKey by user_name_* get user_name_*   alpha

	key := MakeListKey(t.prefix, handler.Channel, handler.Topic)

	for {
		select {
		case <-t.seqDone:
			logger.New().Info("--------Sequential STOP--------")
			return
		case <-t.seqTicker.C:

		}

		// sort will cause Performance issues
		cmd := t.client.Sort(ctx, key, &redis.Sort{
			By: "",
			// Offset: 0,
			// Count:  0,
			Get:   nil,
			Order: "DESC",
			Alpha: true,
		})

		if err := cmd.Err(); err != nil {
			logger.New().With("", err).Error("sort error")
			continue
		}

		vals := cmd.Val()

		if len(vals) > 0 {
			t.doConsumeSeq(ctx, key, handler.Channel, handler.Topic, vals)
		}
	}
}

func (t *scheduleJob) doConsumeSeq(ctx context.Context, key, channel, topic string, vals []string) {

	var msg Message
	xAddArgs := redisx.NewZAddArgs(MakeStreamKey(t.prefix, channel, topic), "", "*", t.maxLen, 0, nil)
	for _, val := range vals {

		strs := strings.SplitN(val, ":", 2)
		if err := t.client.LRem(ctx, key, 1, val).Err(); err != nil {
			logger.New().Error(err)
		}

		if len(strs) < 2 {
			continue
		}
		if err := json.Unmarshal([]byte(strs[1]), &msg); err != nil {
			logger.New().Error(err)
		}
		xAddArgs.Values = map[string]any(msg.Values)
		if err := t.client.XAdd(ctx, xAddArgs).Err(); err != nil {
			logger.New().Error(err)
		}
	}

}

func (t *scheduleJob) shutDown() {
	t.done <- struct{}{}
	t.seqDone <- struct{}{}
}
