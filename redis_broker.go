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
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"beanq/helper/stringx"
	"beanq/internal/base"
	opt "beanq/internal/options"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
)

type Broker interface {
	enqueue(ctx context.Context, stream string, task *Task, options opt.Option) error
	close() error
	start(ctx context.Context, consumers []*ConsumerHandler)
}

type RedisBroker struct {
	client      *redis.Client
	ctx         context.Context
	done, stop  chan struct{}
	healthCheck healthCheckI
	scheduleJob scheduleJobI
	logJob      logJobI
	opts        *opt.Options
	wg          *sync.WaitGroup
	once        *sync.Once
}

var _ Broker = new(RedisBroker)

func NewRedisBroker(config BeanqConfig) *RedisBroker {
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
		ctx:         nil,
		done:        make(chan struct{}),
		stop:        make(chan struct{}),
		healthCheck: newHealthCheck(client),
		scheduleJob: newScheduleJob(client),
		logJob:      newLogJob(client),
		opts:        nil,
		wg:          &sync.WaitGroup{},
		once:        &sync.Once{},
	}
}

func (t *RedisBroker) enqueue(ctx context.Context, stream string, task *Task, opts opt.Option) error {
	if stream == "" || task == nil {
		return fmt.Errorf("stream or values can't empty")
	}
	if err := t.scheduleJob.enqueue(ctx, stream, task, opts); err != nil {
		return err
	}
	return nil
}

func (t *RedisBroker) start(ctx context.Context, consumers []*ConsumerHandler) {
	// it is useless
	p, _ := ants.NewPool(4)
	defer p.Release()

	if opts, ok := ctx.Value("options").(*opt.Options); ok {
		t.opts = opts
	}

	var cancel context.CancelFunc
	t.ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	t.wg.Add(4)
	if err := p.Submit(func() {
		t.wg.Done()
		t.worker(consumers)
	}); err != nil {
		Logger.Error(err)
	}
	// consumer schedule jobs
	if err := p.Submit(func() {
		t.wg.Done()
		t.scheduleJob.start(t.ctx, consumers)
	}); err != nil {
		Logger.Error(err)
	}
	// check client health
	if err := p.Submit(func() {
		t.wg.Done()
		t.healthCheckerStart()
	}); err != nil {
		Logger.Error(err)
	}
	if err := p.Submit(func() {
		t.wg.Done()
		t.waitSignal()
	}); err != nil {
		Logger.Error(err)
	}

	// REFERENCE: https://redis.io/commands/xclaim/
	// monitor other stream pending
	// go t.claim(consumers)

	t.wg.Wait()
	Logger.Info("----START----")

	select {
	case <-t.done:
		Logger.Info("----DONE----")
		return
	}
}

func (t *RedisBroker) healthCheckerStart() {

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.done:
			return
		case <-t.ctx.Done():
			if !errors.Is(t.ctx.Err(), context.Canceled) {
				Logger.Error(t.ctx.Err())
			}
			return
		case <-ticker.C:
			if err := t.healthCheck.start(t.ctx); err != nil {
				Logger.Error(err)
				return
			}
		}
	}
}

func (t *RedisBroker) worker(consumers []*ConsumerHandler) {

	workers := make(chan struct{}, t.opts.MinWorkers)

	for _, v := range consumers {
		// if has bound a group,then continue
		result, err := t.client.XInfoGroups(t.ctx, base.MakeStreamKey(v.Group, v.Queue)).Result()
		if err != nil && err.Error() != "ERR no such key" {
			Logger.Errorf("InfoGroupErr:%s", err.Error())
			continue
		}

		if len(result) < 1 {
			if err := t.createGroup(v.Queue, v.Group); err != nil {
				Logger.Errorf("CreateGroupErr:%+s", err.Error())
				continue
			}
		}

		workers <- struct{}{}
		go t.work(v, workers)
	}
}

func (t *RedisBroker) waitSignal() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGSTOP, syscall.SIGHUP)
	select {
	case sig := <-sigs:
		if sig == syscall.SIGTERM || sig == syscall.SIGINT || sig == syscall.SIGSTOP || sig == syscall.SIGHUP {
			t.once.Do(func() {
				close(t.stop)
				t.done <- struct{}{}
			})
			return
		}
	}
}

func (t *RedisBroker) work(handler *ConsumerHandler, workers chan struct{}) {
	ch, err := t.readGroups(handler.Queue, handler.Group, int64(t.opts.MinWorkers))

	if err != nil {
		Logger.Error(err)
		return
	}

	t.consumer(handler.ConsumerFun, handler.Group, ch)
	<-workers
}

func (t *RedisBroker) createGroup(queue, group string) error {
	cmd := t.client.XGroupCreateMkStream(t.ctx, base.MakeStreamKey(group, queue), group, "0")
	if cmd.Err() != nil && cmd.Err().Error() != "BUSYGROUP Consumer Group name already exists" {
		return cmd.Err()
	}
	return nil
}

func (t *RedisBroker) readGroups(queue, group string, count int64) (<-chan *redis.XStream, error) {
	consumer := uuid.New().String()
	ch := make(chan *redis.XStream)
	go func() {
		for {
			select {
			case <-t.done:
				return
			case <-t.ctx.Done():
				if !errors.Is(t.ctx.Err(), context.Canceled) {
					Logger.Error(t.ctx.Err())
				}
				return
			default:
				streams, err := t.client.XReadGroup(t.ctx, &redis.XReadGroupArgs{
					Group:    group,
					Streams:  []string{base.MakeStreamKey(group, queue), ">"},
					Consumer: consumer,
					Count:    count,
					Block:    0,
				}).Result()

				if err != nil {
					Logger.Errorf("XReadGroupErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					continue
				}
				if len(streams) <= 0 {
					continue
				}

				for _, v := range streams {
					ch <- &v
				}
			}
		}
	}()
	return ch, nil
}

// Please refer to http://www.redis.cn/commands/xclaim.html
func (t *RedisBroker) claim(consumers []*ConsumerHandler) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.done:
			return
		case <-t.ctx.Done():
			if !errors.Is(t.ctx.Err(), context.Canceled) {
				Logger.Error(t.ctx.Err())
			}
			return
		case <-ticker.C:
			start := "-"
			end := "+"

			for _, consumer := range consumers {
				res, err := t.client.XPendingExt(t.ctx, &redis.XPendingExtArgs{
					Stream: base.MakeStreamKey(consumer.Group, consumer.Queue),
					Group:  consumer.Group,
					Start:  start,
					End:    end,
					// Count:  10,
				}).Result()
				if err != nil && err != redis.Nil {
					Logger.Errorf("XPendingErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					break
				}
				for _, v := range res {

					if v.Idle.Seconds() > 60 {

						claims, err := t.client.XClaim(t.ctx, &redis.XClaimArgs{

							Stream:   base.MakeStreamKey(consumer.Group, consumer.Queue),
							Group:    consumer.Group,
							Consumer: consumer.Queue,
							MinIdle:  60 * time.Second,

							Messages: []string{v.ID},
						}).Result()
						if err != nil && err != redis.Nil {
							Logger.Errorf("XClaimErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
							continue
						}
						ch := make(chan *redis.XStream, 1)
						ch <- &redis.XStream{
							Stream:   base.MakeStreamKey(consumer.Group, consumer.Queue),
							Messages: claims,
						}
						t.consumer(consumer.ConsumerFun, consumer.Group, ch)
						close(ch)
					}
				}
			}
		}
	}
}

func (t *RedisBroker) consumer(f DoConsumer, group string, ch <-chan *redis.XStream) {
	info := SuccessInfo
	result := &ConsumerResult{
		Level:   InfoLevel,
		Info:    info,
		RunTime: "",
	}
	var now time.Time

	for {
		select {
		case <-t.done:

			return
		case <-t.ctx.Done():
			if !errors.Is(t.ctx.Err(), context.Canceled) {
				Logger.Error(t.ctx.Err())
			}
			return
		case msg := <-ch:

			stream := msg.Stream
			for _, vm := range msg.Messages {

				task := t.parseMapToTask(vm, stream)
				now = time.Now()

				// if error,then retry to consume
				err := base.Retry(func() error {
					return f(task)
				}, t.opts.RetryTime)
				if err != nil {
					info = FailedInfo
					result.Level = ErrLevel
					result.Info = FlagInfo(err.Error())
				}

				sub := time.Now().Sub(now)

				result.Payload = task.Payload()
				result.RunTime = sub.String()
				result.Queue = msg.Stream
				result.Group = group
				// Successfully consumed data, stored in `string`
				if err := t.logJob.saveLog(t.ctx, result); err != nil {
					Logger.Error(err)
					continue
				}

				// `stream` confirmation message
				if err := t.client.XAck(t.ctx, base.MakeStreamKey(group, msg.Stream), group, vm.ID).Err(); err != nil {
					Logger.Errorf("XACKErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					continue
				}
				// delete data from `stream`
				if err := t.client.XDel(t.ctx, base.MakeStreamKey(group, msg.Stream), vm.ID).Err(); err != nil {
					Logger.Errorf("XdelErr:%s,Stack:%v", err.Error(), stringx.ByteToString(debug.Stack()))
					continue
				}
			}
		}
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

func (t *RedisBroker) parseMapToTask(msg redis.XMessage, stream string) *Task {
	payload, id, streamStr, addTime, queue, group, executeTime, retry, maxLen := openTaskMap(BqMessage(msg), stream)
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
	}
}
