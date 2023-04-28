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

	"beanq/helper/json"
	"beanq/helper/timex"
	"beanq/internal/base"
	"beanq/internal/options"
	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"go.uber.org/zap"
)

type (
	scheduleJobI interface {
		start(ctx context.Context, consumer *ConsumerHandler) error
		enqueue(ctx context.Context, task *Task, option options.Option) error
		shutDown()
		sendToStream(ctx context.Context, task *Task) error
	}
	scheduleJob struct {
		client     *redis.Client
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
		scoreMin:       "0",
		scoreMax:       "10",
		offset:         0,
		count:          -1,
		delayJobTicker: 10 * time.Second,
		consumeTicker:  80 * time.Millisecond,
	}
)

func newScheduleJob(pool *ants.Pool, client *redis.Client) *scheduleJob {
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

func (t *scheduleJob) enqueue(ctx context.Context, task *Task, opt options.Option) error {

	bt, err := json.Marshal(task.Values)
	if err != nil {
		return err
	}

	if err := t.client.ZAdd(ctx, base.MakeZSetKey(Config.Queue.Redis.Prefix, opt.Group, opt.Queue), redis.Z{
		Score:  cast.ToFloat64(task.ExecuteTime().Unix()) + opt.Priority,
		Member: bt,
	}).Err(); err != nil {
		return err
	}

	return nil
}

func (t *scheduleJob) consume(ctx context.Context, consumer *ConsumerHandler) {

	ticker := time.NewTicker(defaultScheduleJobConfig.consumeTicker)
	defer ticker.Stop()

	var (
		now, now2 time.Time
	)
	for {
		select {
		case <-ctx.Done():
			t.pool.Release()
			return
		case <-t.done:
			return
		case <-ticker.C:

			now = time.Now()
			now2 = timex.HalfHour(now)
			sub := now2.Sub(now)

			if sub.Seconds() <= 0 {
				sub = 30 * time.Minute
				now2 = now2.Add(30 * time.Minute)
			}

			if err := t.doConsume(ctx, now2, consumer); err != nil {
				Logger.Error("consume err", zap.Error(err))
				continue
			}

			ticker.Reset(sub)
		}
	}
}

func (t *scheduleJob) doConsume(ctx context.Context, time2 time.Time, consumer *ConsumerHandler) error {

	max := time2.Add(9 * time.Second).Unix()

	zRangeBy := &redis.ZRangeBy{
		Min:    defaultScheduleJobConfig.scoreMin,
		Max:    cast.ToString(max),
		Offset: defaultScheduleJobConfig.offset,
		Count:  defaultScheduleJobConfig.count,
	}
	key := base.MakeZSetKey(Config.Queue.Redis.Prefix, consumer.Group, consumer.Queue)
	cmd := t.client.ZRangeByScore(ctx, key, zRangeBy)
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

	doTask := func(ctx context.Context, vv string, consumer *ConsumerHandler) error {
		task, err := jsonToTask(vv)
		if err != nil {
			return err
		}

		if err := t.sendToStream(ctx, task); err != nil {
			return err
		}

		// Delete data from `zset`
		if err := t.client.ZRem(ctx, base.MakeZSetKey(Config.Queue.Redis.Prefix, consumer.Group, consumer.Queue), vv).Err(); err != nil {
			return err
		}
		return nil
	}
	// begin to execute consumer's datas
	for _, vv := range vals {
		if err := doTask(ctx, vv, consumer); err != nil {
			Logger.Error("consumer err", zap.Error(err))
			continue
		}
	}
}

func (t *scheduleJob) sendToStream(ctx context.Context, task *Task) error {
	queue := task.Queue()
	maxLen := task.MaxLen()

	xAddArgs := &redis.XAddArgs{
		Stream:     base.MakeStreamKey(Config.Queue.Redis.Prefix, task.Group(), queue),
		NoMkStream: false,
		MaxLen:     maxLen,
		MinID:      "",
		Approx:     false,
		// Limit:      0,
		ID:     "*",
		Values: map[string]any(task.Values),
	}
	return t.client.XAdd(ctx, xAddArgs).Err()
}

func (t *scheduleJob) shutDown() {
	t.stop <- struct{}{}
	t.done <- struct{}{}
}
