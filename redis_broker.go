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
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"beanq/helper/timex"
	"beanq/internal/base"
	opt "beanq/internal/options"
	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type (
	Broker interface {
		enqueue(ctx context.Context, task *Task, options opt.Option) error
		close() error
		start(ctx context.Context, consumers []*ConsumerHandler)
	}

	RedisBroker struct {
		client      *redis.Client
		isStop      bool
		healthCheck healthCheckI
		scheduleJob scheduleJobI
		logJob      logJobI
		cond        *sync.Cond
		opts        *opt.Options
		wg          *sync.WaitGroup
		pool        *ants.Pool
	}
)

var (
	_ Broker = (*RedisBroker)(nil)
)

func NewRedisBroker(pool *ants.Pool, config BeanqConfig) *RedisBroker {

	client := redis.NewClient(&redis.Options{
		Addr:         config.Queue.Redis.Host + ":" + config.Queue.Redis.Port,
		Password:     config.Queue.Redis.Password,
		DB:           config.Queue.Redis.Database,
		MaxRetries:   config.Queue.Redis.MaxRetries,
		DialTimeout:  config.Queue.Redis.DialTimeout,
		ReadTimeout:  config.Queue.Redis.ReadTimeout,
		WriteTimeout: config.Queue.Redis.WriteTimeout,
		PoolSize:     config.Queue.Redis.PoolSize,
		MinIdleConns: config.Queue.Redis.MinIdleConnections,
		PoolTimeout:  config.Queue.Redis.PoolTimeout,
	})
	return &RedisBroker{
		client:      client,
		isStop:      false,
		healthCheck: newHealthCheck(client),
		scheduleJob: newScheduleJob(pool, client),
		logJob:      newLogJob(client),
		cond:        sync.NewCond(&sync.Mutex{}),
		opts:        nil,
		wg:          &sync.WaitGroup{},
		pool:        pool,
	}
}

func (t *RedisBroker) enqueue(ctx context.Context, task *Task, opts opt.Option) error {
	if task == nil {
		return fmt.Errorf("enqueue Task Err:%+v", "stream or values is nil")
	}
	nowTime := timex.HalfHour(time.Now())

	if task.ExecuteTime().Before(nowTime.Add(time.Duration(task.Priority()) * time.Second)) {

		xAddArgs := &redis.XAddArgs{
			Stream:     base.MakeStreamKey(Config.Queue.Redis.Prefix, task.Group(), task.Queue()),
			NoMkStream: false,
			MaxLen:     task.MaxLen(),
			MinID:      "",
			Approx:     false,
			// Limit:      0,
			ID:     "*",
			Values: map[string]any(task.Values),
		}

		if err := t.client.XAdd(ctx, xAddArgs).Err(); err != nil {
			return err
		}
		return nil
	}
	if err := t.scheduleJob.enqueue(ctx, task, opts); err != nil {
		return err
	}
	return nil
}

func (t *RedisBroker) start(ctx context.Context, consumers []*ConsumerHandler) {

	if opts, ok := ctx.Value("options").(*opt.Options); ok {
		t.opts = opts
	}

	for _, consumer := range consumers {

		// consume data
		go t.worker(ctx, consumer)

		// check information
		t.scheduleJob.start(ctx, consumer, t.cond, t.isStop)

	}

	// check client health
	if err := t.healthCheckerStart(ctx); err != nil {
		Logger.Error("health check err", zap.Error(err))
	}
	// REFERENCE: https://redis.io/commands/xclaim/
	// monitor other stream pending
	// go t.claim(ctx, consumers)

	Logger.Info("----START----")
	// // monitor signal
	t.waitSignal()
}

func (t *RedisBroker) healthCheckerStart(ctx context.Context) error {

	do := func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ctx.Done():
				if !errors.Is(ctx.Err(), context.Canceled) {
					Logger.Error("context closed", zap.Error(ctx.Err()))
				}
				ticker.Stop()
				return
			case <-ticker.C:
				if err := t.healthCheck.start(ctx); err != nil {
					Logger.Error("health check", zap.Error(err))
					return
				}
			}
		}
	}
	t.cond.L.Lock()
	for t.isStop {
		t.cond.Wait()
	}
	if err := t.pool.Submit(do); err != nil {
		return err
	}
	t.cond.L.Unlock()
	return nil
}

func (t *RedisBroker) worker(ctx context.Context, consumer *ConsumerHandler) {

	t.cond.L.Lock()
	for t.isStop {
		t.cond.Wait()
	}
	result, err := t.client.XInfoGroups(ctx, base.MakeStreamKey(Config.Queue.Redis.Prefix, consumer.Group, consumer.Queue)).Result()
	if err != nil && err.Error() != "ERR no such key" {
		Logger.Error("worker err", zap.Error(err))
		return
	}

	if len(result) < 1 {
		if err := t.createGroup(ctx, consumer.Queue, consumer.Group); err != nil {
			Logger.Error("worker err", zap.Error(err))
			return
		}
	}

	if err := t.pool.Submit(func() {
		t.work(ctx, 10, consumer)
	}); err != nil {
		Logger.Error("worker err", zap.Error(err))
		return
	}

	t.cond.L.Unlock()
}

func (t *RedisBroker) waitSignal() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGTSTP)

	for {
		select {
		case <-sigs:

			// todo
			// will improve what stop schedule job
			t.isStop = true
			t.pool.Release()
			Logger.Fatal("----STOP----")

		default:
			t.isStop = false
			t.cond.Broadcast()
		}
	}
}

func (t *RedisBroker) createGroup(ctx context.Context, queue, group string) error {
	if err := t.client.XGroupCreateMkStream(ctx, base.MakeStreamKey(Config.Queue.Redis.Prefix, group, queue), group, "0").Err(); err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (t *RedisBroker) work(ctx context.Context, count int64, handler *ConsumerHandler) {

	group := handler.Group
	queue := handler.Queue

	for {
		// block XReadGroup to read data
		readGroup := t.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Streams:  []string{base.MakeStreamKey(Config.Queue.Redis.Prefix, group, queue), ">"},
			Consumer: base.MakeStreamKey(Config.Queue.Redis.Prefix, group, queue),
			Count:    count,
			Block:    10 * time.Second,
		})
		if err := readGroup.Err(); err != nil && err != redis.Nil {
			Logger.Error("XReadGroup err", zap.Error(err))
			continue
		}

		streams, err := readGroup.Result()
		if err != nil {
			continue
		}
		if len(streams) <= 0 {
			continue
		}
		t.consumer(ctx, handler.ConsumerFun, group, streams)
	}
}

// Please refer to http://www.redis.cn/commands/xclaim.html
func (t *RedisBroker) claim(ctx context.Context, consumers []*ConsumerHandler) {
	ticker := time.NewTicker(50 * time.Second)
	defer ticker.Stop()

	streams := make([]redis.XStream, 1)

	for {
		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				Logger.Error("context closed", zap.Error(ctx.Err()))
			}
			return
		case <-ticker.C:
			start := "-"
			end := "+"

			for _, consumer := range consumers {
				res, err := t.client.XPendingExt(ctx, &redis.XPendingExtArgs{
					Stream: base.MakeStreamKey(Config.Queue.Redis.Prefix, consumer.Group, consumer.Queue),
					Group:  consumer.Group,
					Start:  start,
					End:    end,
					// Count:  10,
				}).Result()
				if err != nil && err != redis.Nil {
					Logger.Error("XPending err", zap.Error(err))
					break
				}
				for _, v := range res {

					if v.Idle.Seconds() > 60 {

						claims, err := t.client.XClaim(ctx, &redis.XClaimArgs{

							Stream:   base.MakeStreamKey(Config.Queue.Redis.Prefix, consumer.Group, consumer.Queue),
							Group:    consumer.Group,
							Consumer: consumer.Queue,
							MinIdle:  60 * time.Second,

							Messages: []string{v.ID},
						}).Result()
						if err != nil && err != redis.Nil {
							Logger.Error("XClaim err", zap.Error(err))
							continue
						}

						streams = append(streams, redis.XStream{Stream: base.MakeStreamKey(Config.Queue.Redis.Prefix, consumer.Group, consumer.Queue), Messages: claims})
						t.consumer(ctx, consumer.ConsumerFun, consumer.Group, streams)
						streams = nil
					}
				}
			}
		}
	}
}

func (t *RedisBroker) consumer(ctx context.Context, f DoConsumer, group string, streams []redis.XStream) {
	info := SuccessInfo
	result := &ConsumerResult{
		Level:   InfoLevel,
		Info:    info,
		RunTime: "",
	}
	var now time.Time
	for _, v := range streams {
		stream := v.Stream
		for _, vv := range v.Messages {
			task, err := t.parseMapToTask(vv, stream)
			if err != nil {
				Logger.Error("parse json to task err", zap.Error(err))
				continue
			}
			now = time.Now()

			if task.ExecuteTime().After(now) {
				if err := t.scheduleJob.sendToStream(ctx, task); err != nil {
					Logger.Error("xadd error", zap.Error(err))
				}
			} else {

				// if error,then retry to consume
				if err := base.Retry(func() error {
					return f(task)
				}, t.opts.RetryTime); err != nil {
					info = FailedInfo
					result.Level = ErrLevel
					result.Info = FlagInfo(err.Error())
				}

				sub := time.Now().Sub(now)

				result.Payload = task.Payload()
				result.RunTime = sub.String()
				result.Queue = stream
				result.Group = group
				// Successfully consumed data, stored in `string`
				if err := t.logJob.saveLog(ctx, result); err != nil {
					Logger.Error("save log err", zap.Error(err))
					continue
				}
			}

			// `stream` confirmation message
			if err := t.client.XAck(ctx, base.MakeStreamKey(Config.Queue.Redis.Prefix, group, stream), group, vv.ID).Err(); err != nil {
				Logger.Error("xack err", zap.Error(err))
				continue
			}
			// delete data from `stream`
			if err := t.client.XDel(ctx, base.MakeStreamKey(Config.Queue.Redis.Prefix, group, stream), vv.ID).Err(); err != nil {
				Logger.Error("xdel err", zap.Error(err))
				continue
			}
		}
	}
}

func (t *RedisBroker) close() error {
	return t.client.Close()
}

func (t *RedisBroker) parseMapToTask(msg redis.XMessage, stream string) (*Task, error) {
	payload, id, streamStr, addTime, queue, group, executeTime, retry, maxLen, err := openTaskMap(BqMessage(msg), stream)
	if err != nil {
		return nil, err
	}
	return &Task{
		Values: values{
			"id":          id,
			"name":        streamStr,
			"queue":       queue,
			"group":       group,
			"maxLen":      maxLen,
			"retry":       retry,
			"payload":     payload,
			"addTime":     addTime,
			"executeTime": executeTime,
		},
		rw: new(sync.RWMutex),
	}, nil
}
