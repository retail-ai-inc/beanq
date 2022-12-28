package beanq

import (
	"context"
	"fmt"
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
}

var _ scheduleJobI = new(scheduleJob)

const (
	offset, count int64 = 0, 10
)

func newScheduleJob(client *redis.Client) *scheduleJob {
	return &scheduleJob{client: client}
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

					task := ParseTask([]byte(val))

					if task.ExecuteTime().After(time.Now()) {
						if err := t.client.RPush(ctx, key, val).Err(); err != nil {
							if err != redis.Nil {
								fmt.Println(err.Error())
							}
						}
					} else {
						maps := base.ParseArgs(task.Id(), task.Queue(), task.Name(), task.Payload(), task.Group(), task.Retry(), task.priority, task.MaxLen(), task.ExecuteTime())

						if err := t.enqueue(ctx, base.MakeZSetKey(task.Group(), task.Queue()), maps, options.Option{
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
					task := ParseTask(byteV)

					executeTime := task.ExecuteTime()
					queue := task.Queue()
					maxLen := task.MaxLen()
					values := base.ParseArgs(task.Id(), task.Queue(), task.Name(), task.Payload(), task.Group(), task.Retry(), task.priority, task.MaxLen(), task.ExecuteTime())

					if executeTime.Before(time.Now()) {
						cmd := t.client.XAdd(ctx, &redis.XAddArgs{
							Stream:     base.MakeStreamKey(task.Group(), queue),
							NoMkStream: false,
							MaxLen:     maxLen,
							MinID:      "",
							Approx:     false,
							// Limit:      0,
							ID:     "*",
							Values: values,
						})
						if cmd.Err() != nil {
							fmt.Println(cmd.Err())
						}
					} else {
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
	}
}
