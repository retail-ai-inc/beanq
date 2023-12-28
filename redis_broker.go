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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"github.com/retail-ai-inc/beanq/helper/stringx"
)

type (
	Broker interface {
		enqueue(ctx context.Context, msg *Message, options Option) error
		close() error
		start(ctx context.Context, consumers []*ConsumerHandler)
	}

	RedisBroker struct {
		client                *redis.Client
		done, stop, claimDone chan struct{}
		scheduleJob           scheduleJobI
		logJob                logJobI
		opts                  *Options
		once                  *sync.Once
		pool                  *ants.Pool
	}
)

var _ Broker = (*RedisBroker)(nil)

func NewRedisBroker(pool *ants.Pool, config BeanqConfig) *RedisBroker {

	client := redis.NewClient(&redis.Options{
		Addr:         strings.Join([]string{config.Redis.Host, config.Redis.Port}, ":"),
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
		client:      client,
		done:        make(chan struct{}),
		stop:        make(chan struct{}),
		claimDone:   make(chan struct{}),
		scheduleJob: newScheduleJob(pool, client),
		logJob:      newLogJob(client),
		opts:        nil,
		once:        &sync.Once{},
		pool:        pool,
	}
}

func (t *RedisBroker) enqueue(ctx context.Context, msg *Message, opts Option) error {
	if msg == nil {
		return fmt.Errorf("enqueue Message Err:%+v", "stream or values is nil")
	}

	if msg.ExecuteTime().Before(time.Now()) {

		xAddArgs := redisx.NewZAddArgs(MakeStreamKey(Config.Redis.Prefix, msg.Channel(), msg.Topic()), "", "*", Config.Redis.MaxLen, 0, map[string]any(msg.Values))
		if err := t.client.XAdd(ctx, xAddArgs).Err(); err != nil {
			return err
		}
		return nil
	}
	if err := t.scheduleJob.enqueue(ctx, msg, opts); err != nil {
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
			logger.New().With("", err).Error("worker err")
		}
		// check information
		if err := t.scheduleJob.start(ctx, cs); err != nil {
			logger.New().With("", err).Error("schedule job err")
		}
		// REFERENCE: https://redis.io/commands/xclaim/
		// monitor other stream pending
		if err := t.claim(ctx, cs); err != nil {
			logger.New().With("", err).Error("claim job err")
		}
		consumers[key] = nil
	}

	logger.New().Info("----START----")
	// // monitor signal
	t.waitSignal()
}

func (t *RedisBroker) worker(ctx context.Context, consumer *ConsumerHandler) error {

	result, err := t.client.XInfoGroups(ctx, MakeStreamKey(Config.Redis.Prefix, consumer.Channel, consumer.Topic)).Result()
	if err != nil && err.Error() != "ERR no such key" {
		return err
	}

	if len(result) < 1 {
		if err := t.createGroup(ctx, consumer.Topic, consumer.Channel); err != nil {
			return err
		}
	}

	if err := t.pool.Submit(func() {
		t.work(ctx, Config.MinWorkers, consumer)
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
				t.done <- struct{}{}
				t.claimDone <- struct{}{}
				t.pool.Release()
				t.scheduleJob.shutDown()
				_ = t.client.Close()
			})
		}
	}

}

func (t *RedisBroker) createGroup(ctx context.Context, topic, channel string) error {
	if err := t.client.XGroupCreateMkStream(ctx, MakeStreamKey(Config.Redis.Prefix, channel, topic), channel, "0").Err(); err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (t *RedisBroker) work(ctx context.Context, count int64, handler *ConsumerHandler) {
	// consumer := uuid.New().String()
	channel := handler.Channel
	topic := handler.Topic
	stream := MakeStreamKey(Config.Redis.Prefix, channel, topic)
	readGroupArgs := redisx.NewReadGroupArgs(channel, stream, []string{stream, ">"}, count, 10*time.Second)
	for {
		select {
		case <-t.done:
			logger.New().Info("--------Main Task STOP--------")
			return
		case <-ctx.Done():
			logger.New().Info("--------STOP--------")
			return
		default:

			// block XReadGroup to read data
			streams, err := t.client.XReadGroup(ctx, readGroupArgs).Result()

			if err != nil && err != redis.Nil {
				logger.New().With("", err).Error("XReadGroup err")
				continue
			}

			if len(streams) <= 0 {
				continue
			}
			t.consumer(ctx, handler.ConsumerFun, channel, streams)
		}
	}
}

// Please refer to http://www.redis.cn/commands/xclaim.html
func (t *RedisBroker) claim(ctx context.Context, consumer *ConsumerHandler) error {

	return t.pool.Submit(func() {

		streamKey := MakeStreamKey(Config.Redis.Prefix, consumer.Channel, consumer.Topic)
		xAutoClaim := redisx.NewAutoClaimArgs(streamKey, consumer.Channel, 50*time.Second, "0-0", 100, consumer.Topic)

		ticker := time.NewTicker(100 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				if !errors.Is(ctx.Err(), context.Canceled) {
					logger.New().With("", ctx.Err()).Error("context closed")
				}
				return
			case <-t.claimDone:
				logger.New().Info("--------Claim STOP--------")
				return
			case <-ticker.C:

				var streams []redis.XStream

				claims, _, err := t.client.XAutoClaim(ctx, xAutoClaim).Result()

				if err != nil && err != redis.Nil {
					logger.New().With("", err).Error("XClaim err")
					continue
				}

				if len(claims) > 0 {
					streams = append(streams, redis.XStream{Stream: streamKey, Messages: claims})
					t.consumer(ctx, consumer.ConsumerFun, consumer.Channel, streams)
					streams = nil
				}
			}
		}
	})

}

var result = sync.Pool{New: func() any {
	return &ConsumerResult{
		Level:   InfoLevel,
		Info:    SuccessInfo,
		RunTime: "",
	}
}}

func (t *RedisBroker) consumer(ctx context.Context, f DoConsumer, channel string, streams []redis.XStream) {

	for key, v := range streams {

		stream := v.Stream
		message := v.Messages

		for _, vv := range message {
			msg, err := t.parseMapToMessage(vv, stream)
			if err != nil {
				logger.New().With("", err).Error("parse json to Message err")
				continue
			}
			r := result.Get().(*ConsumerResult)
			r.Id = vv.ID
			r.BeginTime = time.Now()
			// if error,then retry to consume
			nerr := make(chan error, 1)
			if err := RetryInfo(func() error {
				defer func() {
					if ne := recover(); ne != nil {
						nerr <- fmt.Errorf("error:%+v,stack:%s", ne, stringx.ByteToString(debug.Stack()))
					}
				}()
				return f(msg)
			}, t.opts.JobMaxRetry); err != nil {
				nerr <- err
			}
			select {
			case v := <-nerr:
				if v != nil {
					r.Level = ErrLevel
					r.Info = FlagInfo(v.Error())
				}
			default:

			}
			r.EndTime = time.Now()

			sub := r.EndTime.Sub(r.BeginTime)

			r.Payload = msg.Payload()
			r.RunTime = sub.String()
			r.ExecuteTime = msg.ExecuteTime()
			r.Topic = stream
			r.Channel = channel
			// Successfully consumed data, stored in `string`
			if err := t.logJob.saveLog(ctx, r); err != nil {
				logger.New().With("", err).Error("save log err")
			}

			r = &ConsumerResult{Level: InfoLevel, Info: SuccessInfo, RunTime: ""}
			result.Put(r)

			// `stream` confirmation message
			if err := t.client.XAck(ctx, stream, channel, vv.ID).Err(); err != nil {
				logger.New().With("", err).Error("xack err")
			}
			// delete data from `stream`
			if err := t.client.XDel(ctx, stream, vv.ID).Err(); err != nil {
				logger.New().With("", err).Error("xdel err")
			}
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

func (t *RedisBroker) parseMapToMessage(msg redis.XMessage, stream string) (*Message, error) {
	message, id, streamStr, addTime, topic, channel, executeTime, retry, maxLen, err := openMessageMap(BqMessage(msg), stream)
	if err != nil {
		return nil, err
	}
	return &Message{
		Values: values{
			"id":          id,
			"name":        streamStr,
			"topic":       topic,
			"channel":     channel,
			"maxLen":      maxLen,
			"retry":       retry,
			"message":     message,
			"addTime":     addTime,
			"executeTime": executeTime,
		},
	}, nil
}
