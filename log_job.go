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
	"fmt"
	"runtime/debug"
	"time"

	"beanq/helper/json"
	"beanq/helper/stringx"
	"beanq/helper/timex"
	"beanq/internal/base"
	opt "beanq/internal/options"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type (
	FlagInfo string
	LevelMsg string

	ConsumerResult struct {
		Level   LevelMsg
		Info    FlagInfo
		Payload any

		AddTime                string
		RunTime                string
		BeginTime              time.Time
		EndTime                time.Time
		Queue, Group, Consumer string
	}

	logJobI interface {
		saveLog(ctx context.Context, result *ConsumerResult) error
		archive(ctx context.Context) error
	}

	logJob struct {
		client *redis.Client
	}
)

const (
	SuccessInfo FlagInfo = "success"
	FailedInfo  FlagInfo = "failed"

	ErrLevel  LevelMsg = "error"
	InfoLevel LevelMsg = "info"
)

func newLogJob(client *redis.Client) *logJob {
	return &logJob{client: client}
}

func (t *logJob) setEx(ctx context.Context, key string, val []byte, expiration time.Duration) error {
	return t.client.SetEx(ctx, key, val, expiration).Err()
}

func (t *logJob) saveLog(ctx context.Context, result *ConsumerResult) error {
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
	// default ErrorLevel
	uuids := uuid.NewString()
	key := base.MakeLogKey(Config.Queue.Redis.Prefix, "fail", uuids)
	expiration := opts.KeepFailedJobsInHistory
	// InfoLevel
	if result.Level == InfoLevel {
		key = base.MakeLogKey(Config.Queue.Redis.Prefix, "success", uuids)
		expiration = opts.KeepSuccessJobsInHistory
	}
	return t.setEx(ctx, key, b, expiration)
}

func (t *logJob) archive(ctx context.Context) error {
	return nil
}
