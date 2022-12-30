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
)

type scheduleJobI interface {
	start(ctx context.Context, consumers []*ConsumerHandler)
	enqueue(ctx context.Context, zsetStr string, values map[string]any, option options.Option) error
}

type scheduleJob struct {
	client *redis.Client
	wg     *sync.WaitGroup
}

var _ scheduleJobI = new(scheduleJob)

const (
	offset, count int64 = 0, 10
)

func newScheduleJob(client *redis.Client) *scheduleJob {
	return &scheduleJob{client: client, wg: &sync.WaitGroup{}}
}

func (t *scheduleJob) start(ctx context.Context, consumers []*ConsumerHandler) {

	go t.delayJobs(ctx, consumers)
	go t.consume(ctx, consumers)

}
func (t *scheduleJob) enqueue(ctx context.Context, zsetStr string, values map[string]any, opt options.Option) error {

	if values == nil {
		return fmt.Errorf("values can't empty")
	}

	bt, err := json.Marshal(values)
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
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, consumer := range consumers {
				key := base.MakeListKey(consumer.Group, consumer.Queue)
				cmd := t.client.LRange(ctx, key, offset, count)
				if cmd.Err() != nil && cmd.Err() != redis.Nil {
					fmt.Println(cmd.Err().Error())
					continue
				}
				values := cmd.Val()
				if len(values) <= 0 {
					continue
				}
				for _, val := range values {

					task := jsonToTask([]byte(val))

					flag := false
					if task.ExecuteTime().After(time.Now()) {
						flag = true
					}
					// if delay job
					if flag {
						if err := t.client.RPush(ctx, key, val).Err(); err != nil {
							if err != redis.Nil {
								fmt.Println(err.Error())
							}
						}
					}
					// if not delay job
					if !flag {
						maps := makeTaskMap(task.id, task.queue, task.name, task.Payload(), task.group, task.retry, task.priority, task.maxLen, task.executeTime)

						if err := t.enqueue(ctx, base.MakeZSetKey(task.group, task.queue), maps, options.Option{
							Priority: task.priority,
						}); err != nil {

						}
					}
					if err := t.client.LRem(ctx, key, 1, val).Err(); err != nil {
						fmt.Println(err.Error())
					}
				}
			}
		}
	}
}
func (t *scheduleJob) consume(ctx context.Context, consumers []*ConsumerHandler) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

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
			continue
		}
		val := cmd.Val()

		if len(val) <= 0 {
			continue
		}

		for _, vv := range val {

			byteV := []byte(vv)
			task := jsonToTask(byteV)

			executeTime := task.executeTime

			flag := false
			if executeTime.Before(time.Now()) {
				flag = true
			}

			if flag {
				if err := t.sendToStream(ctx, task); err != nil {
					// handle err
				}
			}
			if !flag {
				// if executeTime after now
				if err := t.client.LPush(ctx, base.MakeListKey(consumer.Group, consumer.Queue), vv).Err(); err != nil {
					fmt.Println(err)
					continue
				}
			}
			if err := t.client.ZRem(ctx, base.MakeZSetKey(consumer.Group, consumer.Queue), vv).Err(); err != nil {
				fmt.Println(err)
				continue
			}
		}
	}
}
func (t *scheduleJob) sendToStream(ctx context.Context, task Task) error {

	queue := task.queue
	maxLen := task.maxLen
	values := makeTaskMap(task.id, task.queue, task.name, task.Payload(), task.group, task.retry, task.priority, task.maxLen, task.executeTime)

	xaddArgs := &redis.XAddArgs{
		Stream:     base.MakeStreamKey(task.group, queue),
		NoMkStream: false,
		MaxLen:     maxLen,
		MinID:      "",
		Approx:     false,
		// Limit:      0,
		ID:     "*",
		Values: values,
	}
	cmd := t.client.XAdd(ctx, xaddArgs)
	return cmd.Err()

}
