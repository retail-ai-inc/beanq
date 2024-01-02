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
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/retail-ai-inc/beanq/helper/timex"
)

type (
	FlagInfo string
	LevelMsg string

	ConsumerResult struct {
		Id      string
		Level   LevelMsg
		Info    FlagInfo
		Payload any

		AddTime                  string
		ExpireTime               time.Time
		RunTime                  string
		BeginTime                time.Time
		EndTime                  time.Time
		ExecuteTime              time.Time
		Topic, Channel, Consumer string
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
	var opts *Options
	if optsVal, ok := ctx.Value("options").(*Options); ok {
		opts = optsVal
	}
	now := time.Now()
	if result.AddTime == "" {
		result.AddTime = now.Format(timex.DateTime)
	}

	// default ErrorLevel

	key := strings.Join([]string{MakeLogKey(Config.Redis.Prefix, "fail"), result.Id}, ":")
	expiration := opts.KeepFailedJobsInHistory

	// InfoLevel
	if result.Level == InfoLevel {
		key = strings.Join([]string{MakeLogKey(Config.Redis.Prefix, "success"), result.Id}, ":")
		expiration = opts.KeepSuccessJobsInHistory
	}
	result.ExpireTime = time.UnixMilli(now.UnixMilli() + expiration.Milliseconds())

	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("JsonMarshalErr:%s,Stack:%+v", err.Error(), stringx.ByteToString(debug.Stack()))
	}
	return t.client.Set(ctx, key, b, Config.KeepSuccessJobsInHistory).Err()
}

func (t *logJob) archive(ctx context.Context) error {
	// TODO
	return nil
}
