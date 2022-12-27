package beanq

import (
	"context"
	"fmt"
	"time"

	"beanq/helper/json"
	"beanq/helper/stringx"
	"beanq/internal/options"
	"github.com/go-redis/redis/v8"
)

type scheduleJobI interface {
	start(ctx context.Context, consumers []*ConsumerHandler)
	enqueue(ctx context.Context, values map[string]any, option options.Option) error
}
type scheduleJob struct {
	client *redis.Client
}

var _ scheduleJobI = new(scheduleJob)

func newScheduleJob(client *redis.Client) *scheduleJob {
	return &scheduleJob{client: client}
}
func (t *scheduleJob) start(ctx context.Context, consumers []*ConsumerHandler) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	jn := json.Json
	for {
		select {
		case <-ticker.C:

			for _, v := range consumers {

				cmd := t.client.ZRevRange(ctx, v.Queue+"-list", 0, 10)
				if cmd.Err() != nil {
					continue
				}
				val := cmd.Val()
				if len(val) <= 0 {
					continue
				}
				fmt.Printf("%+v \n", val)
				for _, vv := range val {
					vvtobyte := stringx.StringToByte(vv)
					executeTimeStr := jn.Get(vvtobyte, "executeTime")
					//executeTime := cast.ToTime(executeTimeStr)
					fmt.Printf("执行时间：%s \n", executeTimeStr)
					//fmt.Println(executeTime)
				}

				//t.client.XAdd(ctx, &redis.XAddArgs{
				//	Stream:     opts.Queue,
				//	NoMkStream: false,
				//	MaxLen:     opts.MaxLen,
				//	MinID:      "",
				//	Approx:     false,
				//	// Limit:      0,
				//	ID:     "*",
				//	Values: values,
				//})
			}

		}
	}
}
func (t *scheduleJob) enqueue(ctx context.Context, values map[string]any, opt options.Option) error {

	var stream string
	if streamVal, ok := values["queue"]; ok {
		if v, ok := streamVal.(string); ok {
			stream = v
		}
	}

	data, err := json.Json.MarshalToString(values)
	if err != nil {
		return err
	}

	if err := t.client.ZAdd(ctx, stream, &redis.Z{
		Score:  opt.Priority,
		Member: data,
	}).Err(); err != nil {
		return err
	}

	return nil
}
