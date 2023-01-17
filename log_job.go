package beanq

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"beanq/helper/json"
	"beanq/helper/stringx"
	"beanq/helper/timex"
	opt "beanq/internal/options"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type FlagInfo string
type LevelMsg string

const (
	SuccessInfo FlagInfo = "success"
	FailedInfo  FlagInfo = "failed"

	ErrLevel  LevelMsg = "error"
	InfoLevel LevelMsg = "info"
)

type ConsumerResult struct {
	Level   LevelMsg
	Info    FlagInfo
	Payload any

	AddTime string
	RunTime string

	Queue, Group, Consumer string
}

type logJobI interface {
	success(ctx context.Context, result *ConsumerResult) error
	fail(ctx context.Context) error
	archive(ctx context.Context) error
}
type logJob struct {
	client *redis.Client
}

func newLogJob(client *redis.Client) *logJob {
	return &logJob{client: client}
}
func (t *logJob) success(ctx context.Context, result *ConsumerResult) error {
	var opts *opt.Options
	if optsVal, ok := ctx.Value("options").(*opt.Options); ok {
		opts = optsVal
	}

	if result.AddTime == "" {
		result.AddTime = time.Now().Format(timex.DateTime)
	}
	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("JsonMarshalErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
	}

	if err := t.client.SetEX(ctx, "result:success:"+uuid.NewString(), b, opts.KeepSuccessJobsInHistory).Err(); err != nil {
		Logger.Error(err)
	}
	return nil
}
func (t *logJob) fail(ctx context.Context) error {
	return nil
}
func (t *logJob) archive(ctx context.Context) error {
	return nil
}
