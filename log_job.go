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

	"github.com/go-redis/redis/v8"
	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/retail-ai-inc/beanq/helper/timex"
	"github.com/spf13/cast"
)

type (
	FlagInfo string
	LevelMsg string

	ConsumerResult struct {
		Id      string
		Level   LevelMsg
		Info    FlagInfo
		Payload any

		PendingRetry             int64
		Retry                    int
		Priority                 float64
		AddTime                  string
		ExpireTime               time.Time
		RunTime                  string
		BeginTime                time.Time
		EndTime                  time.Time
		ExecuteTime              time.Time
		Topic, Channel, Consumer string
		MoodType                 string
	}

	ILogJob interface {
		saveLog(ctx context.Context, result *ConsumerResult) error
		expire(ctx context.Context, done <-chan struct{})
		archive(ctx context.Context) error
	}

	logJob struct {
		client            redis.UniversalClient
		pool              *ants.Pool
		prefix            string
		expiration        time.Duration
		expirationSuccess time.Duration
	}
)

func (c *ConsumerResult) MarshalBinary() (data []byte, err error) {
	return json.Marshal(c)
}

func (c *ConsumerResult) Initialize() *ConsumerResult {
	c.Level = InfoLevel
	c.Info = SuccessInfo
	c.RunTime = ""
	return c
}

const (
	SuccessInfo FlagInfo = "success"
	FailedInfo  FlagInfo = "failed"

	ErrLevel  LevelMsg = "error"
	InfoLevel LevelMsg = "info"
)

func newLogJob(config *BeanqConfig, client redis.UniversalClient, pool *ants.Pool) *logJob {
	return &logJob{
		client:            client,
		pool:              pool,
		prefix:            config.Redis.Prefix,
		expiration:        config.KeepFailedJobsInHistory,
		expirationSuccess: config.KeepSuccessJobsInHistory,
	}
}

func (t *logJob) setEx(ctx context.Context, key string, val []byte, expiration time.Duration) error {
	return t.client.SetEX(ctx, key, val, expiration).Err()
}

func (t *logJob) saveLog(ctx context.Context, result *ConsumerResult) error {

	now := time.Now()
	if result.AddTime == "" {
		result.AddTime = now.Format(timex.DateTime)
	}

	// default ErrorLevel

	key := strings.Join([]string{MakeLogKey(t.prefix, "fail")}, ":")
	expiration := t.expiration

	// InfoLevel
	if result.Level == InfoLevel {
		key = strings.Join([]string{MakeLogKey(t.prefix, "success")}, ":")
		expiration = t.expirationSuccess
	}

	result.ExpireTime = time.UnixMilli(now.UnixMilli() + expiration.Milliseconds())

	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("JsonMarshalErr:%s,Stack:%+v", err.Error(), stringx.ByteToString(debug.Stack()))
	}

	return t.client.ZAdd(ctx, key, &redis.Z{
		Score:  float64(result.ExpireTime.UnixMilli()),
		Member: b,
	}).Err()

}

func (t *logJob) expire(ctx context.Context, done <-chan struct{}) {

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	failKey := MakeLogKey(t.prefix, "fail")
	successKey := MakeLogKey(t.prefix, "success")

	for {
		// check state
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
		}

		if err := t.pool.Submit(func() {
			t.client.ZRemRangeByScore(ctx, failKey, "0", cast.ToString(time.Now().UnixMilli()))
		}); err != nil {
			logger.New().Error(err)
		}

		if err := t.pool.Submit(func() {
			t.client.ZRemRangeByScore(ctx, successKey, "0", cast.ToString(time.Now().UnixMilli()))
		}); err != nil {
			logger.New().Error(err)
		}
	}

}

func (t *logJob) archive(ctx context.Context) error {
	// TODO
	return nil
}
