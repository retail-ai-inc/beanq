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
	enqueue(ctx context.Context, stream string, values map[string]any, option options.Option) error
}
type scheduleJob struct {
	client *redis.Client
}

var _ scheduleJobI = new(scheduleJob)

func newScheduleJob(client *redis.Client) *scheduleJob {
	return &scheduleJob{client: client}
}
func (t *scheduleJob) start(ctx context.Context, consumers []*ConsumerHandler) {
	go t.delayJobs(ctx, consumers)
	go t.consumSet(ctx, consumers)
}
func (t *scheduleJob) enqueue(ctx context.Context, stream string, values map[string]any, opt options.Option) error {

	if values == nil {
		return fmt.Errorf("values can't empty")
	}

	bt, err := json.Marshal(values)
	if err != nil {
		return err
	}
	if err := t.client.ZAdd(ctx, stream+":zset", &redis.Z{
		Score:  opt.Priority,
		Member: bt,
	}).Err(); err != nil {
		return err
	}

	return nil
}
func (t *scheduleJob) delayJobs(ctx context.Context, consumers []*ConsumerHandler) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, v := range consumers {
				key := v.Group + ":" + v.Queue + ":list"
				cmd := t.client.LPop(ctx, key)
				if cmd.Err() != nil {
					if cmd.Err() != redis.Nil {
						fmt.Println(cmd.Err().Error())
					}

					continue
				}
				if len(cmd.Val()) <= 0 {
					continue
				}

				task := ParseTask([]byte(cmd.Val()))

				if task.ExecuteTime().After(time.Now()) {
					if err := t.client.RPush(ctx, key, cmd.Val()).Err(); err != nil {
						if err != redis.Nil {
							fmt.Println(err.Error())
						}
					}
					continue
				}
				maps := base.ParseArgs(task.Id(), task.Queue(), task.Name(), task.Payload(), task.Group(), task.Retry(), task.priority, task.MaxLen(), task.ExecuteTime())

				t.enqueue(ctx, task.Group()+":"+task.Queue(), maps, options.Option{
					Priority: task.priority,
				})
			}
		}
	}

}
func (t *scheduleJob) consumSet(ctx context.Context, consumers []*ConsumerHandler) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			for _, v := range consumers {
				cmd := t.client.ZRevRangeByScore(ctx, v.Group+":"+v.Queue+":zset", &redis.ZRangeBy{
					Min:    "0",
					Max:    "10",
					Offset: 0,
					Count:  10,
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
							Stream:     task.Group() + ":" + queue + ":stream",
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
						//if executeTime after now
						if err := t.client.LPush(ctx, v.Group+":"+v.Queue+":list", vv).Err(); err != nil {
							fmt.Println(err)
							continue
						}
					}

					if err := t.client.ZRem(ctx, v.Group+":"+v.Queue+":zset", vv).Err(); err != nil {
						fmt.Println(err)
						continue
					}
				}
			}
		}
	}
}