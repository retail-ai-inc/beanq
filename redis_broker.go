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

	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type (
	Broker interface {
		enqueue(ctx context.Context, task *Task, options Option) error
		close() error
		start(ctx context.Context, consumers []*ConsumerHandler)
	}

	RedisBroker struct {
		client                                  *redis.Client
		done, stop, healCheckDone, logCheckDone chan struct{}
		healthCheck                             healthCheckI
		scheduleJob                             scheduleJobI
		logJob                                  logJobI
		opts                                    *Options
		wg                                      *sync.WaitGroup
		once                                    *sync.Once
		pool                                    *ants.Pool
	}
)

var _ Broker = (*RedisBroker)(nil)

func NewRedisBroker(pool *ants.Pool, config BeanqConfig) *RedisBroker {

	client := redis.NewClient(&redis.Options{
		Addr:         config.Redis.Host + ":" + config.Redis.Port,
		Password:     config.Redis.Password,
		DB:           config.Redis.Database,
		MaxRetries:   config.JobMaxRetries,
		DialTimeout:  config.Redis.DialTimeout,
		ReadTimeout:  config.Redis.ReadTimeout,
		WriteTimeout: config.Redis.WriteTimeout,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.Redis.MinIdleConnections,
		PoolTimeout:  config.Redis.PoolTimeout,
	})
	return &RedisBroker{
		client:        client,
		done:          make(chan struct{}),
		stop:          make(chan struct{}),
		healCheckDone: make(chan struct{}),
		logCheckDone:  make(chan struct{}),
		healthCheck:   newHealthCheck(client),
		scheduleJob:   newScheduleJob(pool, client),
		logJob:        newLogJob(client),
		opts:          nil,
		wg:            &sync.WaitGroup{},
		once:          &sync.Once{},
		pool:          pool,
	}
}

func (t *RedisBroker) enqueue(ctx context.Context, task *Task, opts Option) error {
	if task == nil {
		return fmt.Errorf("enqueue Task Err:%+v", "stream or values is nil")
	}

	if task.ExecuteTime().Before(time.Now()) {

		xAddArgs := &redis.XAddArgs{
			Stream:     MakeStreamKey(Config.Redis.Prefix, task.Group(), task.Queue()),
			NoMkStream: false,
			MaxLen:     task.MaxLen(),
			MinID:      "",
			Approx:     false,
			ID:         "*",
			Values:     map[string]any(task.Values),
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

	if opts, ok := ctx.Value("options").(*Options); ok {
		t.opts = opts
	}

	for key, consumer := range consumers {
		cs := consumer
		// consume data
		if err := t.worker(ctx, cs); err != nil {
			Logger.Error("worker err", zap.Error(err))
		}
		// check information
		if err := t.scheduleJob.start(ctx, cs); err != nil {
			Logger.Error("schedule job err", zap.Error(err))
		}
		consumers[key] = nil
	}

	// check client health
	if err := t.healthCheckerStart(ctx); err != nil {
		Logger.Error("health check err", zap.Error(err))
	}
	t.CheckLogs(ctx)

	// REFERENCE: https://redis.io/commands/xclaim/
	// monitor other stream pending
	// go t.claim(ctx, consumers)

	Logger.Info("----START----")
	// // monitor signal
	t.waitSignal()
}
func (t *RedisBroker) CheckLogs(ctx context.Context) {
	if err := t.pool.Submit(func() {
		ticker := time.NewTicker(10 * time.Second)

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-t.logCheckDone:
				ticker.Stop()
				return
			case <-ticker.C:
				t.logJob.checkExpiration(ctx)
			}
		}
	}); err != nil {
		Logger.Error("check logs error:%+v", zap.Error(err))
	}
}
func (t *RedisBroker) healthCheckerStart(ctx context.Context) error {

	if err := t.pool.Submit(func() {

		ticker := time.NewTicker(10 * time.Second)

		for {
			select {
			case <-ctx.Done():
				if !errors.Is(ctx.Err(), context.Canceled) {
					Logger.Error("context closed", zap.Error(ctx.Err()))
				}
				ticker.Stop()
				return
			case <-t.healCheckDone:
				ticker.Stop()
				return
			case <-ticker.C:
				if err := t.healthCheck.start(ctx); err != nil {
					Logger.Error("health check", zap.Error(err))
					return
				}
			}
		}
	}); err != nil {
		return err
	}
	return nil
}

func (t *RedisBroker) worker(ctx context.Context, consumer *ConsumerHandler) error {

	result, err := t.client.XInfoGroups(ctx, MakeStreamKey(Config.Redis.Prefix, consumer.Group, consumer.Queue)).Result()
	if err != nil && err.Error() != "ERR no such key" {
		return err
	}

	if len(result) < 1 {
		if err := t.createGroup(ctx, consumer.Queue, consumer.Group); err != nil {
			return err
		}
	}

	if err := t.pool.Submit(func() {
		t.work(ctx, 10, consumer)
	}); err != nil {
		return err
	}

	return nil
}

func (t *RedisBroker) waitSignal() {
	sigs := make(chan os.Signal)

	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGTSTP)

	select {
	case sig := <-sigs:
		if sig == syscall.SIGINT {
			t.once.Do(func() {
				close(t.stop)
				t.pool.Release()
				t.done <- struct{}{}
				t.healCheckDone <- struct{}{}
				t.logCheckDone <- struct{}{}
				t.scheduleJob.shutDown()
				_ = t.client.Close()
			})
		}
	}

}

func (t *RedisBroker) createGroup(ctx context.Context, queue, group string) error {
	if err := t.client.XGroupCreateMkStream(ctx, MakeStreamKey(Config.Redis.Prefix, group, queue), group, "0").Err(); err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (t *RedisBroker) work(ctx context.Context, count int64, handler *ConsumerHandler) {
	// consumer := uuid.New().String()
	group := handler.Group
	queue := handler.Queue
	stream := MakeStreamKey(Config.Redis.Prefix, group, queue)
	readGroupArgs := &redis.XReadGroupArgs{
		Group:    group,
		Streams:  []string{stream, ">"},
		Consumer: stream,
		Count:    count,
		Block:    10 * time.Second,
	}
	for {
		select {
		case <-t.done:
			Logger.Info("--------STOP--------")
			return
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				Logger.Error("context closed", zap.Error(ctx.Err()))
			}
			return
		default:

			// block XReadGroup to read data
			streams, err := t.client.XReadGroup(ctx, readGroupArgs).Result()

			if err != nil && err != redis.Nil {
				Logger.Error("XReadGroup err", zap.Error(err))
				continue
			}
			if len(streams) <= 0 {
				continue
			}
			t.consumer(ctx, handler.ConsumerFun, group, streams)
		}
	}
}

// Please refer to http://www.redis.cn/commands/xclaim.html
func (t *RedisBroker) claim(ctx context.Context, consumers []*ConsumerHandler) {
	ticker := time.NewTicker(50 * time.Second)
	defer ticker.Stop()

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
				streamKey := MakeStreamKey(Config.Redis.Prefix, consumer.Group, consumer.Queue)
				res, err := t.client.XPendingExt(ctx, &redis.XPendingExtArgs{
					Stream: streamKey,
					Group:  consumer.Group,
					Start:  start,
					End:    end,
					// Count:  10,
				}).Result()
				if err != nil && err != redis.Nil {
					Logger.Error("XPending err", zap.Error(err))
					break
				}
				streams := make([]redis.XStream, len(res))
				for _, v := range res {

					if v.Idle.Seconds() > 60 {

						claims, err := t.client.XClaim(ctx, &redis.XClaimArgs{

							Stream:   streamKey,
							Group:    consumer.Group,
							Consumer: consumer.Queue,
							MinIdle:  60 * time.Second,

							Messages: []string{v.ID},
						}).Result()
						if err != nil && err != redis.Nil {
							Logger.Error("XClaim err", zap.Error(err))
							continue
						}

						streams = append(streams, redis.XStream{Stream: streamKey, Messages: claims})
						t.consumer(ctx, consumer.ConsumerFun, consumer.Group, streams)
						streams = nil
					}
				}
			}
		}
	}
}

var result = sync.Pool{New: func() any {
	return &ConsumerResult{
		Level:   InfoLevel,
		Info:    SuccessInfo,
		RunTime: "",
	}
}}

func (t *RedisBroker) consumer(ctx context.Context, f DoConsumer, group string, streams []redis.XStream) {

	for key, v := range streams {
		stream := v.Stream
		message := v.Messages
		streamKey := MakeStreamKey(Config.Redis.Prefix, group, stream)

		for _, vv := range message {
			task, err := t.parseMapToTask(vv, stream)
			if err != nil {
				Logger.Error("parse json to task err", zap.Error(err))
				continue
			}
			r := result.Get().(*ConsumerResult)
			r.BeginTime = time.Now()
			// if error,then retry to consume
			if err := RetryInfo(func() error {
				return f(task)
			}, t.opts.RetryTime); err != nil {
				r.Level = ErrLevel
				r.Info = FlagInfo(err.Error())
			}

			r.EndTime = time.Now()

			sub := r.EndTime.Sub(r.BeginTime)

			r.Payload = task.Payload()
			r.RunTime = sub.String()
			r.ExecuteTime = task.ExecuteTime()
			r.Queue = stream
			r.Group = group
			// Successfully consumed data, stored in `string`
			if err := t.logJob.saveLog(ctx, r); err != nil {
				Logger.Error("save log err", zap.Error(err))
			}

			r = &ConsumerResult{Level: InfoLevel, Info: SuccessInfo, RunTime: ""}
			result.Put(r)

			// `stream` confirmation message
			if err := t.client.XAck(ctx, streamKey, group, vv.ID).Err(); err != nil {
				Logger.Error("xack err", zap.Error(err))

			}
			// delete data from `stream`
			// if err := t.client.XDel(ctx, MakeStreamKey(Config.Redis.Prefix, group, stream), vv.ID).Err(); err != nil {
			// 	Logger.Error("xdel err", zap.Error(err))
			//
			// }
		}
		streams[key] = redis.XStream{}
	}
}

func (t *RedisBroker) close() error {
	select {
	case <-t.stop:
		return errors.New("redis broker already closed")
	default:
		close(t.stop)
	}
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
