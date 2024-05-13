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
	"golang.org/x/sync/errgroup"
)

type (
	RedisBroker struct {
		client           redis.UniversalClient
		scheduleJob      scheduleJobI
		filter           VolatileLFU
		consumerHandlers []IHandle
		logJob           ILogJob
		once             *sync.Once
		pool             *ants.Pool
		prefix           string
		maxLen           int64
		config           *BeanqConfig
	}
)

func newRedisBroker(config *BeanqConfig, pool *ants.Pool) IBroker {

	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        []string{strings.Join([]string{config.Redis.Host, config.Redis.Port}, ":")},
		Password:     config.Redis.Password,
		DB:           config.Redis.Database,
		MaxRetries:   config.Redis.MaxRetries,
		DialTimeout:  config.Redis.DialTimeout,
		ReadTimeout:  config.Redis.ReadTimeout,
		WriteTimeout: config.Redis.WriteTimeout,
		PoolSize:     config.Redis.PoolSize,
		MinIdleConns: config.Redis.MinIdleConnections,
		PoolTimeout:  config.Redis.PoolTimeout,
		PoolFIFO:     true,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		logger.New().Fatal(err.Error())
	}

	broker := &RedisBroker{
		client: client,
		logJob: newLogJob(config, client, pool),
		once:   &sync.Once{},
		pool:   pool,
		prefix: config.Redis.Prefix,
		maxLen: config.Redis.MaxLen,
		config: config,
		filter: &RedisUnique{client: client, ticker: time.NewTicker(30 * time.Second)},
	}
	broker.scheduleJob = broker.newScheduleJob()

	return broker
}

func (t *RedisBroker) enqueue(ctx context.Context, msg *Message) error {

	b, err := t.filter.Add(ctx, MakeFilter(t.prefix), msg.Id)
	if b {
		return nil
	}
	if err != nil {
		return err
	}

	// Sequential job
	if msg.MoodType == string(SEQUENTIAL) {

		mmsg := messageToMap(msg)

		xAddArgs := redisx.NewZAddArgs(MakeStreamKey(sequentialSubscribe, t.prefix, msg.Channel, msg.Topic), "", "*", t.maxLen, 0, mmsg)
		return t.client.XAdd(ctx, xAddArgs).Err()

	}

	// normal job
	if msg.ExecuteTime.Before(time.Now()) {
		nmsg := messageToMap(msg)
		xAddArgs := redisx.NewZAddArgs(MakeStreamKey(normalSubscribe, t.prefix, msg.Channel, msg.Topic), "", "*", t.maxLen, 0, nmsg)
		if err := t.client.XAdd(ctx, xAddArgs).Err(); err != nil {
			return err
		}
		return nil
	}
	// delay job
	if err := t.scheduleJob.enqueue(ctx, msg); err != nil {
		return err
	}
	return nil
}

func (t *RedisBroker) addConsumer(subType subscribeType, channel, topic string, subscribe IConsumeHandle) {

	bqConfig := t.config
	handler := &RedisHandle{
		broker:           t,
		channel:          channel,
		topic:            topic,
		subscribe:        subscribe,
		subscribeType:    subType,
		deadLetterTicker: time.NewTicker(100 * time.Second),
		pendingIdle:      2 * time.Minute,
		jobMaxRetry:      bqConfig.JobMaxRetries,
		minConsumers:     bqConfig.MinConsumers,
		timeOut:          bqConfig.ConsumeTimeOut,
		wg:               new(sync.WaitGroup),
		result: &sync.Pool{New: func() any {
			return &ConsumerResult{
				Level:   InfoLevel,
				Info:    SuccessInfo,
				RunTime: "",
			}
		}},
		errGroupPool: &sync.Pool{New: func() any {
			group := new(errgroup.Group)
			group.SetLimit(2)
			return group
		}},
		once: sync.Once{},
	}
	t.consumerHandlers = append(t.consumerHandlers, handler)
}

func (t *RedisBroker) newScheduleJob() *scheduleJob {
	return &scheduleJob{
		broker:         t,
		wg:             &sync.WaitGroup{},
		scheduleTicker: time.NewTicker(defaultScheduleJobConfig.consumeTicker),
		seqTicker:      time.NewTicker(10 * time.Second),
		scheduleErrGroupPool: &sync.Pool{New: func() any {
			group := new(errgroup.Group)
			group.SetLimit(2)
			return group
		}},
	}

}

func (t *RedisBroker) startConsuming(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	for key, cs := range t.consumerHandlers {
		// consume data
		if err := t.worker(ctx, cs); err != nil {
			logger.New().With("", err).Error("worker err")
		}

		if err := t.scheduleJob.start(ctx, cs); err != nil {
			logger.New().With("", err).Error("schedule job err")
		}
		// REFERENCE: https://redis.io/commands/xclaim/
		// monitor other stream pending
		if err := t.deadLetter(ctx, cs); err != nil {
			logger.New().With("", err).Error("claim job err")
		}
		t.consumerHandlers[key] = nil
	}
	if err := t.pool.Submit(func() {
		t.logJob.expire(ctx)
	}); err != nil {
		logger.New().Error(err)
	}
	if err := t.pool.Submit(func() {
		t.filter.Delete(ctx, MakeFilter(t.prefix))
	}); err != nil {
		logger.New().Error(err)
	}
	logger.New().Info("----START----")
	// monitor signal
	t.waitSignal(cancel)
}

func (t *RedisBroker) worker(ctx context.Context, handle IHandle) error {
	if err := handle.Check(ctx); err != nil {
		return err
	}
	if err := t.pool.Submit(func() {
		handle.Process(ctx)
	}); err != nil {
		return err
	}

	return nil
}

func (t *RedisBroker) deadLetter(ctx context.Context, handle IHandle) error {

	return t.pool.Submit(func() {
		if err := handle.DeadLetter(ctx); err != nil {
			logger.New().Error(err)
		}
	})
}

func (t *RedisBroker) waitSignal(cancel context.CancelFunc) {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGTSTP)

	select {
	case sig := <-sigs:
		if sig == syscall.SIGINT {
			t.once.Do(func() {
				_ = t.client.Close()
				t.pool.Release()
				cancel()
			})
		}
	}

}

func (t *RedisBroker) close() error {
	return nil
}
