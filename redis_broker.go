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
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
)

type (
	Broker interface {
		enqueue(ctx context.Context, msg *Message, options Option) error
		close() error
		start(ctx context.Context, consumers []*ConsumerHandler)
	}

	RedisBroker struct {
		client                   redis.UniversalClient
		done, claimDone, logDone chan struct{}
		scheduleJob              scheduleJobI
		logJob                   logJobI
		once                     *sync.Once
		pool                     *ants.Pool
		prefix                   string
		maxLen                   int64
	}
)

var _ Broker = (*RedisBroker)(nil)

func newRedisBroker(pool *ants.Pool, config BeanqConfig) *RedisBroker {

	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        []string{strings.Join([]string{config.Redis.Host, config.Redis.Port}, ":")},
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

	prefix := Config.Load().(BeanqConfig).Redis.Prefix
	if prefix == "" {
		prefix = DefaultOptions.Prefix
	}
	maxLen := Config.Load().(BeanqConfig).Redis.MaxLen
	if maxLen <= 0 {
		maxLen = DefaultOptions.DefaultMaxLen
	}

	return &RedisBroker{
		client:      client,
		done:        make(chan struct{}),
		claimDone:   make(chan struct{}),
		logDone:     make(chan struct{}),
		scheduleJob: newScheduleJob(pool, client),
		logJob:      newLogJob(client, pool),
		// opts:        nil,
		once:   &sync.Once{},
		pool:   pool,
		prefix: prefix,
		maxLen: maxLen,
	}
}

func (t *RedisBroker) enqueue(ctx context.Context, msg *Message, opts Option) error {

	if msg == nil {
		return fmt.Errorf("enqueue Message Err:%+v", "stream or values is nil")
	}

	// Sequential job
	if opts.OrderKey != "" {
		if err := t.scheduleJob.sequentialEnqueue(ctx, msg, opts); err != nil {
			return err
		}
		return nil
	}

	// normal job
	if msg.ExecuteTime.Before(time.Now()) {
		nmsg, err := messageToMap(msg)
		if err != nil {
			return err
		}
		xAddArgs := redisx.NewZAddArgs(MakeStreamKey(t.prefix, msg.ChannelName, msg.TopicName), "", "*", t.maxLen, 0, nmsg)
		if err := t.client.XAdd(ctx, xAddArgs).Err(); err != nil {
			return err
		}
		return nil
	}
	// delay job
	if err := t.scheduleJob.enqueue(ctx, msg, opts); err != nil {
		return err
	}
	return nil

}

func (t *RedisBroker) start(ctx context.Context, consumers []*ConsumerHandler) {

	for key, consumer := range consumers {

		cs := consumer
		cs.IHandle = newRedisHandle(t.client, cs.Channel, cs.Topic, cs.ConsumerFun, t.pool)

		// consume data
		if err := t.worker(ctx, cs); err != nil {
			logger.New().With("", err).Error("worker err")
		}

		//
		if err := t.scheduleJob.start(ctx, cs); err != nil {
			logger.New().With("", err).Error("schedule job err")
		}
		// REFERENCE: https://redis.io/commands/xclaim/
		// monitor other stream pending
		if err := t.deadLetter(ctx, cs); err != nil {
			logger.New().With("", err).Error("claim job err")
		}
		consumers[key] = nil
	}
	if err := t.pool.Submit(func() {
		t.logJob.expire(ctx, t.logDone)
	}); err != nil {
		logger.New().Error(err)
	}
	logger.New().Info("----START----")
	// monitor signal
	t.waitSignal()
}

func (t *RedisBroker) worker(ctx context.Context, handle *ConsumerHandler) error {

	if err := handle.Check(ctx); err != nil {
		return err
	}
	if err := t.pool.Submit(func() {
		handle.Work(ctx, t.done)
	}); err != nil {
		return err
	}

	return nil
}

func (t *RedisBroker) deadLetter(ctx context.Context, handle *ConsumerHandler) error {

	return t.pool.Submit(func() {
		if err := handle.DeadLetter(ctx, t.claimDone); err != nil {
			logger.New().Error(err)
		}
	})
}

func (t *RedisBroker) waitSignal() {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGTSTP)

	select {
	case sig := <-sigs:
		if sig == syscall.SIGINT {
			t.once.Do(func() {
				t.done <- struct{}{}
				t.claimDone <- struct{}{}
				t.logDone <- struct{}{}
				t.pool.Release()
				t.scheduleJob.shutDown()
				_ = t.client.Close()
			})
		}
	}

}

func (t *RedisBroker) close() error {
	return nil
}
