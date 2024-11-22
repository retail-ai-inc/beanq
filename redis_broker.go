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
	"errors"
	"fmt"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/retail-ai-inc/beanq/v3/internal/driver/bredis"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/retail-ai-inc/beanq/v3/helper/timex"
	"github.com/spf13/cast"
)

type (
	RedisBroker struct {
		filter             VolatileLFU
		client             redis.UniversalClient
		config             *BeanqConfig
		captureException   func(ctx context.Context, err any)
		logJob             *Log
		once               *sync.Once
		asyncPool          *asyncPool
		consumerHandlerDic sync.Map
		prefix             string
		failKey            string
		successKey         string
		consumerHandlers   []IHandle
		maxLen             int64
	}
)

func (t *RedisBroker) driver() any {
	return t.client
}

func (t *RedisBroker) setCaptureException(handler func(ctx context.Context, err any)) {
	if handler != nil {
		t.captureException = handler
		t.asyncPool.captureException = handler
	}
}

// Archive log
func (t *RedisBroker) Archive(ctx context.Context, result *Message, isSequential bool) error {
	//
	//// log for mongo to batch saving
	//logStream := tool.MakeLogicKey(t.prefix)
	//val := map[string]any{
	//	"id":           result.Id,
	//	"status":       result.Status,
	//	"level":        result.Level,
	//	"info":         result.Info,
	//	"payload":      result.Payload,
	//	"pendingRetry": result.PendingRetry,
	//	"retry":        result.Retry,
	//	"priority":     result.Priority,
	//	"addTime":      result.AddTime,
	//	"runTime":      result.RunTime,
	//	"beginTime":    result.BeginTime,
	//	"endTime":      result.EndTime,
	//	"executeTime":  result.ExecuteTime,
	//	"topic":        result.Topic,
	//	"channel":      result.Channel,
	//	"consumer":     result.Consumer,
	//	"moodType":     result.MoodType,
	//	"response":     result.Response,
	//}
	//
	//if isSequential {
	//	// status saved in redis,6 hour
	//	key := tool.MakeStatusKey(t.prefix, result.Channel, result.Id)
	//	if err := bredis.SaveHSetScript.Run(ctx, t.client, []string{key}, val).Err(); err != nil {
	//		return err
	//	}
	//}
	//
	//// write job log into redis
	//if err := t.client.XAdd(ctx, &redis.XAddArgs{
	//	Stream:     logStream,
	//	NoMkStream: false,
	//	MaxLen:     20000,
	//	Approx:     false,
	//	ID:         "*",
	//	Values:     val,
	//}).Err(); err != nil {
	//	return err
	//}

	return nil
}

// Obsolete log
func (t *RedisBroker) Obsolete(ctx context.Context, data []map[string]any) error {

	timer := timex.TimerPool.Get(5 * time.Second)
	defer timex.TimerPool.Put(timer)

	key := tool.MakeLogicKey(t.prefix)

	for {
		// check state
		select {
		case <-ctx.Done():
			logger.New().Info("Redis Obsolete Stop")
			return nil
		case <-timer.C:
		}
		timer.Reset(5 * time.Second)
		result, err := t.client.XReadGroup(ctx, bredis.NewReadGroupArgs(tool.BeanqLogGroup, key, []string{key, ">"}, 200, 0)).Result()
		if err != nil {
			if strings.Contains(err.Error(), "NOGROUP No such") {
				if err := t.client.XGroupCreateMkStream(ctx, key, tool.BeanqLogGroup, "0").Err(); err != nil {
					t.captureException(ctx, err)
					return nil
				}
				continue
			}
			if errors.Is(err, context.Canceled) {
				logger.New().Info("Redis Obsolete Stop")
				return nil
			}
			if !errors.Is(err, redis.Nil) && !errors.Is(err, redis.ErrClosed) {
				t.captureException(ctx, err)
			}
			continue
		}
		if len(result) <= 0 {
			continue
		}
		messages := result[0].Messages
		datas := make([]map[string]any, 0, len(messages))
		ids := make([]string, 0, len(messages))

		for _, v := range messages {
			if v.ID != "" {
				ids = append(ids, v.ID)
				datas = append(datas, v.Values)
			}
		}

		if err := t.logJob.Obsoletes(ctx, datas); err != nil {
			t.captureException(ctx, err)
			continue
		}
		if _, err := t.client.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
			pipeliner.XAck(ctx, key, tool.BeanqLogGroup, ids...)
			pipeliner.XDel(ctx, key, ids...)
			return nil
		}); err != nil {
			t.captureException(ctx, err)
		}

	}
}

// Delete delete expire id
func (t *RedisBroker) Delete(ctx context.Context, key string) error {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("UniqueId Obsolete Task Stop: %w", ctx.Err())
		case <-ticker.C:
			cmd := t.client.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
				Min:    "-inf",
				Max:    "+inf",
				Offset: 0,
				Count:  100,
			})
			val := cmd.Val()
			if len(val) <= 0 {
				continue
			}

			for _, v := range val {
				floor := math.Floor(v.Score)
				frac := v.Score - floor
				expTime := cast.ToTime(cast.ToInt(floor))

				if time.Since(expTime).Seconds() >= 3600*2 {
					err := t.client.ZRem(ctx, key, v.Member).Err()
					if err != nil {
						t.captureException(ctx, err)
					}
					continue
				}
				if time.Since(expTime).Seconds() >= 60*30 && frac*1000 <= 2 {
					err := t.client.ZRem(ctx, key, v.Member).Err()
					if err != nil {
						t.captureException(ctx, err)
					}
					continue
				}
			}
		}
	}
}

func (t *RedisBroker) startConsuming(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	for key, cs := range t.consumerHandlers {
		cs := cs
		// consume data
		if err := t.worker(ctx, cs); err != nil {
			t.captureException(ctx, err)
		}

		t.asyncPool.Execute(ctx, func(ctx context.Context) error {
			return cs.Schedule(ctx)
		})

		// REFERENCE: https://redis.io/commands/xclaim/
		// monitor other stream pending
		t.asyncPool.Execute(ctx, func(ctx context.Context) error {
			return cs.DeadLetter(ctx)
		})

		t.consumerHandlers[key] = nil
	}
	if t.config.History.On {
		//consume logs
		t.asyncPool.Execute(ctx, func(ctx context.Context) error {
			return t.Obsolete(ctx, nil)
		})
		//if consumption fails,then retry again via dead letter
		t.asyncPool.Execute(ctx, func(c context.Context) error {
			t.logicLogDeadLetter(ctx)
			return nil
		})
	}

	logger.New().Info("Beanq Start")
	// monitor signal
	<-t.waitSignal(cancel)
}

func (t *RedisBroker) logicLogDeadLetter(ctx context.Context) {

	duration := 10 * time.Second
	timer := timex.TimerPool.Get(duration)
	defer timex.TimerPool.Put(timer)
	streamKey := tool.MakeLogicKey(t.prefix)
	minId := "-"

	for {
		select {
		case <-ctx.Done():
			logger.New().Info("Logic Job Stop")
			return
		case <-timer.C:
			timer.Reset(duration)
			cmd := t.client.XPendingExt(ctx, &redis.XPendingExtArgs{
				Stream: streamKey,
				Group:  tool.BeanqLogGroup,
				//Idle parameter need redis6.2.0
				//if message has been reading,but still not complete after 15 minutes,then it is dead letter message
				//Idle:   15 * time.Minute,
				Start: minId,
				End:   "+",
				Count: 100,
			})
			results, err := cmd.Result()
			if err != nil {
				t.captureException(ctx, err)
				continue
			}
			if len(results) <= 0 {
				minId = "-"
				continue
			}
			for _, result := range results {
				minId = result.ID
				if result.Idle < 15*time.Minute {
					continue
				}
				// add lock
				logicLock := tool.MakeLogicLock(t.prefix, result.ID)

				if v := bredis.AddLogicLockScript.Run(ctx, t.client, []string{logicLock}).Val(); v.(int64) == 1 {
					continue
				}

				r, err := t.client.XRangeN(ctx, streamKey, result.ID, result.ID, 1).Result()
				if err != nil {
					t.captureException(ctx, err)
					continue
				}
				if len(r) <= 0 {
					continue
				}
				val := r[0].Values
				pendingRetry := 0
				if v, ok := val["pendingRetry"]; ok {
					pendingRetry = cast.ToInt(v) + 1
				}
				val["pendingRetry"] = pendingRetry

				if err := t.client.Watch(ctx, func(tx *redis.Tx) error {
					if _, err := tx.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
						pipeliner.XAck(ctx, streamKey, tool.BeanqLogGroup, result.ID)
						pipeliner.XDel(ctx, streamKey, result.ID)
						pipeliner.XAdd(ctx, &redis.XAddArgs{
							Stream: streamKey,
							Values: val,
						})
						return nil
					}); err != nil {
						return err
					}
					return nil
				}, streamKey); err != nil {
					t.captureException(ctx, err)
				}
				//release lock
				if err := t.client.Del(ctx, logicLock).Err(); err != nil {
					t.captureException(ctx, err)
				}
			}
		}
	}

}

func (t *RedisBroker) worker(ctx context.Context, handle IHandle) error {
	t.asyncPool.Execute(ctx, func(ctx context.Context) error {
		handle.Process(ctx)
		return nil
	})
	return nil
}

func (t *RedisBroker) waitSignal(cancel context.CancelFunc) <-chan bool {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigs
		cancel()
		_ = t.client.Close()
		t.asyncPool.Release()
		_ = logger.New().Sync()
		done <- true
	}()
	return done
}
