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

var _ scheduleJobI = new(scheduleJob)

const (
	offset, count int64 = 0, 10
)

func newScheduleJob(client *redis.Client) *scheduleJob {
	pool, _ := ants.NewPool(10, ants.WithPreAlloc(true))
	return &scheduleJob{client: client, wg: &sync.WaitGroup{}, pool: pool}
}

func (t *scheduleJob) start(ctx context.Context, consumers []*ConsumerHandler) {

	go t.delayJobs(ctx, consumers)
	go t.consume(ctx, consumers)
}
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

func (t *scheduleJob) delayJobs(ctx context.Context, consumers []*ConsumerHandler) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer func() {
		ticker.Stop()
	}()

	defer t.pool.Release()

	for {
		select {
		case <-ticker.C:
			for _, consumer := range consumers {
				key := base.MakeListKey(consumer.Group, consumer.Queue)

				cmd := t.client.LRange(ctx, key, offset, count)
				if cmd.Err() != nil && cmd.Err() != redis.Nil {
					Logger.Error(cmd.Err())
					continue
				}
				vals := cmd.Val()
				if len(vals) <= 0 {
					continue
				}
				t.doDelayJobs(ctx, key, vals)
			}
		}
	}
}
func (t *scheduleJob) doDelayJobs(ctx context.Context, key string, vals []string) {

	for _, val := range vals {
		t.wg.Add(1)
		err := t.pool.Submit(func() {
			defer t.wg.Done()
			task := jsonToTask([]byte(val))

			flag := false
			if task.ExecuteTime().After(time.Now()) {
				flag = true
			}
			// if delay job
			if flag {
				if err := t.client.RPush(ctx, key, val).Err(); err != nil {

					Logger.Error(err)
					return

				}
			}
			// if not delay job
			if !flag {
				if err := t.enqueue(ctx, base.MakeZSetKey(task.Group(), task.Queue()), task, options.Option{
					Priority: task.Priority(),
				}); err != nil {
					Logger.Error(err)
					return
				}
			}
			if err := t.client.LRem(ctx, key, 1, val).Err(); err != nil {
				Logger.Error(err)
				return
			}
		})
		if err != nil {
			Logger.Error(err)
			continue
		}
	}
	t.wg.Wait()
}
func (t *scheduleJob) consume(ctx context.Context, consumers []*ConsumerHandler) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			t.doConsume(ctx, consumers)
		}
	}
}

func (t *scheduleJob) doConsume(ctx context.Context, consumers []*ConsumerHandler) {

	for _, consumer := range consumers {

		cmd := t.client.ZRevRangeByScore(ctx, base.MakeZSetKey(consumer.Group, consumer.Queue), &redis.ZRangeBy{
			Min:    "0",
			Max:    "10",
			Offset: offset,
			Count:  count,
		})
		if cmd.Err() != nil {
			Logger.Error(cmd.Err())
			continue
		}
		val := cmd.Val()

		if len(val) <= 0 {
			continue
		}

		for _, vv := range val {

			byteV := []byte(vv)
			task := jsonToTask(byteV)
			executeTime := task.ExecuteTime()

			flag := false
			if executeTime.Before(time.Now()) {
				flag = true
			}

			if flag {
				if err := t.sendToStream(ctx, task); err != nil {
					Logger.Error(err)
					continue
				}
			}
			if !flag {
				// if executeTime after now
				if err := t.client.LPush(ctx, base.MakeListKey(consumer.Group, consumer.Queue), vv).Err(); err != nil {
					Logger.Error(err)
					continue
				}
			}
			if err := t.client.ZRem(ctx, base.MakeZSetKey(consumer.Group, consumer.Queue), vv).Err(); err != nil {
				Logger.Error(err)
				continue
			}
		}
	}
}
func (t *scheduleJob) sendToStream(ctx context.Context, task *Task) error {

	queue := task.Queue()
	maxLen := task.MaxLen()

	xaddArgs := &redis.XAddArgs{
		Stream:     base.MakeStreamKey(task.Group(), queue),
		NoMkStream: false,
		MaxLen:     maxLen,
		MinID:      "",
		Approx:     false,
		// Limit:      0,
		ID:     "*",
		Values: map[string]any(task.Values),
	}
	cmd := t.client.XAdd(ctx, xaddArgs)
	return cmd.Err()

}
