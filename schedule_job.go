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

// Package beanq
// @Description:
package beanq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"beanq/helper/json"
	"beanq/internal/base"
	"beanq/internal/options"
	"github.com/go-redis/redis/v8"
	"github.com/panjf2000/ants/v2"
)

type scheduleJobI interface {
	start(ctx context.Context, consumers []*ConsumerHandler)
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
	// ants pool size
	poolSize int
	// zset attribute score
	scoreMin, scoreMax string
	// zset data limit
	offset, count int64
	// delayJob and consumer executeTime
	delayJobTimeSetup             []int64
	delayJobTicker, consumeTicker time.Duration
}{
	poolSize:          10,
	scoreMin:          "0",
	scoreMax:          "10",
	offset:            0,
	count:             500,
	delayJobTimeSetup: []int64{5, 10, 100, 1000, 5000},
	delayJobTicker:    40 * time.Millisecond,
	consumeTicker:     80 * time.Millisecond,
}

func newScheduleJob(client *redis.Client) *scheduleJob {
	pool, _ := ants.NewPool(defaultScheduleJobConfig.poolSize, ants.WithPreAlloc(true))
	return &scheduleJob{client: client, wg: &sync.WaitGroup{}, pool: pool}
}

func (t *scheduleJob) start(ctx context.Context, consumers []*ConsumerHandler) {

	go t.delayJobs(ctx, consumers)
	go t.consume(ctx, consumers)
}

// enqueue
//
//	@Description:
//
// publish []byte data to zset
//
//	@receiver t
//	@param ctx
//	@param zsetStr
//	@param task
//	@param opt
//	@return error
func (t *scheduleJob) enqueue(ctx context.Context, zsetStr string, task *Task, opt options.Option) error {

	if task == nil {
		return fmt.Errorf("values can't empty")
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

// delayJobs
//
//	@Description:
//
// Poll `list`
//
//	@receiver t
//	@param ctx
//	@param consumers

func (t *scheduleJob) delayJobs(ctx context.Context, consumers []*ConsumerHandler) {

	ticker := time.NewTicker(defaultScheduleJobConfig.delayJobTicker)
	defer func() {
		ticker.Stop()
		t.pool.Release()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			for _, consumer := range consumers {
				key := base.MakeListKey(consumer.Group, consumer.Queue)
				// get a data in the `list` header
				cmd := t.client.LPop(ctx, key)
				if cmd.Err() != nil && cmd.Err() != redis.Nil {
					Logger.Error(cmd.Err())
					continue
				}
				vals := cmd.Val()

				if vals == "" {
					continue
				}
				t.doDelayJobs(ctx, key, vals)
			}
		}
	}
}
func (t *scheduleJob) doDelayJobs(ctx context.Context, key string, vals string) {
	// delay task
	doTask := func(ctx context.Context, key, val string) error {
		task := jsonToTask([]byte(val))

		flag := false
		if task.ExecuteTime().After(time.Now()) {
			flag = true
		}
		// if delay job,publish the data to the end of `list`
		if flag {
			if err := t.client.RPush(ctx, key, val).Err(); err != nil {
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
	if err := doTask(ctx, key, vals); err != nil {
		Logger.Error(err)
		return
	}

}

// consume
//
//	@Description:
//
// consume data
//
//	@receiver t
//	@param ctx
//	@param consumers
func (t *scheduleJob) consume(ctx context.Context, consumers []*ConsumerHandler) {
	ticker := time.NewTicker(defaultScheduleJobConfig.consumeTicker)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.doConsume(ctx, consumers)
		}
	}
}

// doConsume
//
//	@Description:
//
// Get data from `zset`, through score attribute
//
//	@receiver t
//	@param ctx
//	@param consumers
func (t *scheduleJob) doConsume(ctx context.Context, consumers []*ConsumerHandler) {

	var wg sync.WaitGroup
	var newConsumer *ConsumerHandler

	for _, consumer := range consumers {
		key := base.MakeZSetKey(consumer.Group, consumer.Queue)
		cmd := t.client.ZRevRangeByScore(ctx, key, &redis.ZRangeBy{
			Min:    defaultScheduleJobConfig.scoreMin,
			Max:    defaultScheduleJobConfig.scoreMax,
			Offset: defaultScheduleJobConfig.offset,
			Count:  defaultScheduleJobConfig.count,
		})
		if cmd.Err() != nil {
			Logger.Error(cmd.Err())
			continue
		}
		val := cmd.Val()
		if len(val) <= 0 {
			continue
		}

		wg.Add(1)
		newConsumer = consumer
		if err := t.pool.Submit(func() {
			defer wg.Done()
			t.doConsumeZset(ctx, val, newConsumer)
		}); err != nil {
			Logger.Error(err)
			continue
		}
	}
	wg.Wait()
}

// doConsumeZset
//
//	@Description:
//	Consumption `zset` data
//	@receiver t
//	@param ctx
//	@param vals
//	@param consumer
func (t *scheduleJob) doConsumeZset(ctx context.Context, vals []string, consumer *ConsumerHandler) {

	doTask := func(ctx context.Context, vv string, consumer *ConsumerHandler) error {
		byteV := []byte(vv)
		task := jsonToTask(byteV)
		executeTime := task.ExecuteTime()

		flag := false
		if executeTime.Before(time.Now()) {
			flag = true
		}
		// if need to consume now,then send data to `stream`
		if flag {
			if err := t.sendToStream(ctx, task); err != nil {
				return err
			}
		}
		// If the message is delayed, it will be pushed to the `list` header
		if !flag {
			// if executeTime after now
			if err := t.client.LPush(ctx, base.MakeListKey(consumer.Group, consumer.Queue), vv).Err(); err != nil {
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
			Logger.Error(err)
			continue
		}
	}

}

// sendToStream
//
//	@Description:
//
// send datas to stream
//
//	@receiver t
//	@param ctx
//	@param task
//	@return error
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
