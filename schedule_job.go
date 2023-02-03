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
	"errors"
	"sync"
	"time"

	"beanq/helper/json"
	"beanq/internal/base"
	"beanq/internal/options"
	"github.com/go-redis/redis/v8"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

type scheduleJobI interface {
	start(ctx context.Context, consumers []*ConsumerHandler) error
	enqueue(ctx context.Context, zsetStr string, task *Task, option options.Option) error
}

type scheduleJob struct {
	client *redis.Client
	wg     *sync.WaitGroup
	pool   *ants.Pool
}

var _ scheduleJobI = (*scheduleJob)(nil)

// schedule job config
var defaultScheduleJobConfig = struct {
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
	count:          500,
	delayJobTicker: 10 * time.Second,
	consumeTicker:  80 * time.Millisecond,
}

func newScheduleJob(pool *ants.Pool, client *redis.Client) *scheduleJob {

	return &scheduleJob{client: client, wg: &sync.WaitGroup{}, pool: pool}
}

func (t *scheduleJob) start(ctx context.Context, consumers []*ConsumerHandler) error {
	if err := t.pool.Submit(func() {
		t.delayJobs(ctx, consumers)
	}); err != nil {
		return err
	}
	if err := t.pool.Submit(func() {
		t.consume(ctx, consumers)
	}); err != nil {
		return err
	}
	return nil
}

func (t *scheduleJob) enqueue(ctx context.Context, zsetStr string, task *Task, opt options.Option) error {
	if task == nil {
		return errors.New("values can't empty")
	}

	bt, err := json.Marshal(task.Values)
	if err != nil {
		return err
	}

	if err := t.client.ZAdd(ctx, zsetStr, &redis.Z{
		Score:  opt.Priority,
		Member: bt,
	}).Err(); err != nil {
		return err
	}

	return nil
}

func (t *scheduleJob) delayJobs(ctx context.Context, consumers []*ConsumerHandler) {
	key := ""
	for _, consumer := range consumers {
		key = base.MakeListKey(consumer.Group, consumer.Queue)

		fun := func() {
			t.pollList(ctx, t.client, key)
		}

		if err := t.pool.Submit(fun); err != nil {
			Logger.Error("poll list err ", zap.Error(err))
			continue
		}
	}
}
func (t *scheduleJob) pollList(ctx context.Context, client *redis.Client, key string) {
	for {
		select {
		case <-ctx.Done():
			t.pool.Release()
			return
		default:
			// get a data from `list` header
			cmd := client.BLPop(ctx, defaultScheduleJobConfig.delayJobTicker, key)
			if cmd.Err() != nil && cmd.Err() != redis.Nil {
				Logger.Error("blpop err", zap.Error(cmd.Err()))
				continue
			}
			vals := cmd.Val()
			if len(vals) < 2 {
				continue
			}
			t.doDelayJobs(ctx, key, vals[1])
		}
	}
}

func (t *scheduleJob) doDelayJobs(ctx context.Context, key string, vals string) {
	// declare delayTask function
	doTask := func(ctx context.Context, client *redis.Client, key, val string) error {
		task := jsonToTask([]byte(val))

		flag := false
		if task.ExecuteTime().After(time.Now()) {
			flag = true
		}
		// if delay job,publish the data to the end of `list`
		if flag {
			if err := client.RPush(ctx, key, val).Err(); err != nil {
				return err
			}
		}
		// if not delay job,send data to zset
		if !flag {
			if err := t.enqueue(ctx, base.MakeZSetKey(task.Group(), task.Queue()), task, options.Option{
				Priority: task.Priority(),
			}); err != nil {
				return err
			}
		}

		return nil
	}
	// begin to execute the task
	if err := doTask(ctx, t.client, key, vals); err != nil {
		Logger.Error("delay job err", zap.Error(err))
		return
	}
}

func (t *scheduleJob) consume(ctx context.Context, consumers []*ConsumerHandler) {
	ticker := time.NewTicker(defaultScheduleJobConfig.consumeTicker)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ctx.Done():
			t.pool.Release()
			return
		case <-ticker.C:
			t.doConsume(ctx, consumers)
		}
	}
}

func (t *scheduleJob) doConsume(ctx context.Context, consumers []*ConsumerHandler) {

	zRangeBy := &redis.ZRangeBy{
		Min:    defaultScheduleJobConfig.scoreMin,
		Max:    defaultScheduleJobConfig.scoreMax,
		Offset: defaultScheduleJobConfig.offset,
		Count:  defaultScheduleJobConfig.count,
	}

	for _, consumer := range consumers {
		key := base.MakeZSetKey(consumer.Group, consumer.Queue)
		cmd := t.client.ZRevRangeByScore(ctx, key, zRangeBy)
		if cmd.Err() != nil {
			Logger.Error("ZRevRangeByScore err", zap.Error(cmd.Err()))
			continue
		}
		val := cmd.Val()
		if len(val) <= 0 {
			continue
		}

		t.doConsumeZset(ctx, val, consumer)
	}
}

func (t *scheduleJob) doConsumeZset(ctx context.Context, vals []string, consumer *ConsumerHandler) {

	doTask := func(ctx context.Context, vv string, consumer *ConsumerHandler) error {
		byteV := []byte(vv)
		task := jsonToTask(byteV)
		executeTime := task.ExecuteTime()

		flag := false
		if executeTime.Before(time.Now()) {
			flag = true
		}
		// if you need to consume now,then send data to `stream`
		if flag {
			if err := t.sendToStream(ctx, task); err != nil {
				return err
			}
		}
		// If the message is delayed, it will be pushed to the `list` header
		if !flag {
			// if executeTime after now
			if err := t.client.RPush(ctx, base.MakeListKey(consumer.Group, consumer.Queue), vv).Err(); err != nil {
				return err
			}
		}
		// Delete data from `zset`
		if err := t.client.ZRem(ctx, base.MakeZSetKey(consumer.Group, consumer.Queue), vv).Err(); err != nil {
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
		Stream:     base.MakeStreamKey(task.Group(), queue),
		NoMkStream: false,
		MaxLen:     maxLen,
		MinID:      "",
		Approx:     false,
		// Limit:      0,
		ID:     "*",
		Values: map[string]any(task.Values),
	}
	cmd := t.client.XAdd(ctx, xAddArgs)
	return cmd.Err()
}
