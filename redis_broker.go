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
	"math"
	"math/rand"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/retail-ai-inc/beanq/helper/timex"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

type (
	RedisBroker struct {
		client              redis.UniversalClient
		scheduleJob         scheduleJobI
		filter              VolatileLFU
		consumerHandlers    []IHandle
		logJob              *Log
		once                *sync.Once
		pool                *ants.Pool
		prefix              string
		maxLen              int64
		config              *BeanqConfig
		failKey, successKey string
	}
)

func newRedisBroker(config *BeanqConfig, pool *ants.Pool) IBroker {

	ctx := context.Background()

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

	if err := client.Ping(ctx).Err(); err != nil {
		logger.New().Fatal(err.Error())
	}

	broker := &RedisBroker{

		client:     client,
		once:       &sync.Once{},
		pool:       pool,
		prefix:     config.Redis.Prefix,
		maxLen:     config.MaxLen,
		config:     config,
		failKey:    MakeLogKey(config.Redis.Prefix, "fail"),
		successKey: MakeLogKey(config.Redis.Prefix, "success"),
	}
	var logs []ILog
	logs = append(logs, broker)
	if config.History.On {
		mongoLog := NewMongoLog(ctx, config)
		logs = append(logs, mongoLog)
	}
	broker.logJob = NewLog(pool, logs...)
	broker.filter = broker
	broker.scheduleJob = broker.newScheduleJob()

	return broker
}

// Archive log
func (t *RedisBroker) Archive(ctx context.Context, result *ConsumerResult) error {
	now := time.Now()
	if result.AddTime == "" {
		result.AddTime = now.Format(timex.DateTime)
	}

	// default ErrorLevel
	key := strings.Join([]string{t.failKey}, ":")
	expiration := t.config.KeepFailedJobsInHistory

	// InfoLevel
	if result.Level == InfoLevel {
		key = strings.Join([]string{t.successKey}, ":")
		expiration = t.config.KeepSuccessJobsInHistory
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

// Obsolete log
func (t *RedisBroker) Obsolete(ctx context.Context) {

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		// check state
		select {
		case <-ctx.Done():
			logger.New().Info("Redis Obsolete Stop")
			return
		case <-ticker.C:
		}

		// delete fail logs
		if err := t.pool.Submit(func() {
			t.client.ZRemRangeByScore(ctx, t.failKey, "0", cast.ToString(time.Now().UnixMilli()))
		}); err != nil {
			logger.New().Error(err)
		}
		// delete success logs
		if err := t.pool.Submit(func() {
			t.client.ZRemRangeByScore(ctx, t.successKey, "0", cast.ToString(time.Now().UnixMilli()))
		}); err != nil {
			logger.New().Error(err)
		}
	}
}

// Add unique id
func (t *RedisBroker) Add(ctx context.Context, key, member string) (bool, error) {
	incr := 0.000
	b := false
	err := t.client.ZRank(ctx, key, member).Err()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			now := time.Now().Unix()
			incr = float64(now) + 0.001
		}
		return b, t.client.ZIncrBy(ctx, key, incr, member).Err()
	}
	incr = 0.001
	b = true
	return b, t.client.ZIncrBy(ctx, key, incr, member).Err()
}

// Delete delete expire id
func (t *RedisBroker) Delete(ctx context.Context, key string) {

	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-ctx.Done():
			logger.New().Info("UniqueId Obsolete Task Stop")
			return
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
					t.client.ZRem(ctx, key, v.Member)
					continue
				}
				if time.Since(expTime).Seconds() >= 60*30 && frac*1000 <= 2 {
					t.client.ZRem(ctx, key, v.Member)
					continue
				}
			}
		}
	}
}

func (t *RedisBroker) checkStatus(ctx context.Context, channel, topic string, id string) (string, error) {
	stringCmd := t.client.Get(ctx, strings.Join([]string{t.prefix, channel, topic, "status", id}, ":"))
	if stringCmd.Err() != nil {
		if errors.Is(stringCmd.Err(), redis.Nil) {
			return "", nil
		}
		return "", stringCmd.Err()
	}
	return stringCmd.Val(), nil

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
		xAddArgs := redisx.NewZAddArgs(MakeStreamKey(sequentialSubscribe, t.prefix, msg.Channel, msg.Topic), "", "*", t.maxLen, 0, msg.ToMap())
		err := t.client.XAdd(ctx, xAddArgs).Err()
		if err != nil {
			return fmt.Errorf("[RedisBroker.enqueue] seq xadd error:%w", err)
		}
		return nil
	}

	// normal job
	if msg.ExecuteTime.Before(time.Now()) {
		xAddArgs := redisx.NewZAddArgs(MakeStreamKey(normalSubscribe, t.prefix, msg.Channel, msg.Topic), "", "*", t.maxLen, 0, msg.ToMap())
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
		pendingIdle:      20 * time.Minute,
		jobMaxRetry:      bqConfig.JobMaxRetries,
		minConsumers:     bqConfig.MinConsumers,
		timeOut:          bqConfig.ConsumeTimeOut,
		wg:               new(sync.WaitGroup),
		resultPool: &sync.Pool{New: func() any {
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

		_ = t.logJob.Obsoletes(ctx)

	}); err != nil {
		logger.New().Error(err)
	}
	if err := t.pool.Submit(func() {
		t.filter.Delete(ctx, MakeFilter(t.prefix))
	}); err != nil {
		logger.New().Error(err)
	}
	logger.New().Info("Beanq Start")
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

func (t *RedisBroker) NewMutex(name string, options ...MuxOption) *Mutex {
	pools := []redis.UniversalClient{
		t.client,
	}
	m := &Mutex{
		name:   name,
		expiry: 8 * time.Second,
		tries:  32,
		delayFunc: func(tries int) time.Duration {
			return time.Duration(rand.Intn(maxRetryDelayMilliSec-minRetryDelayMilliSec)+minRetryDelayMilliSec) * time.Millisecond
		},
		genValueFunc:  genValue,
		driftFactor:   0.01,
		timeoutFactor: 0.05,
		quorum:        len(pools)/2 + 1,
		pools:         pools,
	}

	for _, o := range options {
		o.Apply(m)
	}

	if m.shuffle {
		rand.Shuffle(len(pools), func(i, j int) {
			pools[i], pools[j] = pools[j], pools[i]
		})
	}
	return m
}

func (t *RedisBroker) close() error {
	return nil
}
