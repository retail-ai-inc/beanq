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
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/retail-ai-inc/beanq/helper/timex"
	"github.com/rs/xid"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

type (
	RedisBroker struct {
		client              redis.UniversalClient
		scheduleJob         scheduleJobI
		filter              VolatileLFU
		consumerHandlerDic  sync.Map
		consumerHandlers    []IHandle
		logJob              *Log
		once                *sync.Once
		asyncPool           *asyncPool
		prefix              string
		maxLen              int64
		config              *BeanqConfig
		failKey, successKey string

		captureException func(ctx context.Context, err any)
	}
)

var ErrorIdempotent = errors.New("duplicate id")

func newRedisBroker(config *BeanqConfig) IBroker {

	ctx := context.Background()

	hosts := strings.Split(config.Redis.Host, ",")
	for i, h := range hosts {
		hs := strings.Split(h, ":")
		if len(hs) == 1 {
			hosts[i] = strings.Join([]string{h, config.Redis.Port}, ":")
		}
	}

	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        hosts,
		Password:     config.Redis.Password,
		DB:           config.Redis.Database,
		MaxRetries:   config.Redis.MaxRetries,
		DialTimeout:  config.Redis.DialTimeout,
		ReadTimeout:  config.Redis.ReadTimeout,
		WriteTimeout: config.Redis.WriteTimeout,
		PoolSize:     config.Redis.PoolSize,
		MinIdleConns: config.Redis.MinIdleConnections,
		PoolTimeout:  config.Redis.PoolTimeout,
		PoolFIFO:     false,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		logger.New().Fatal(err.Error())
	}

	asyncPool := newAsyncPool(config.ConsumerPoolSize)

	broker := &RedisBroker{
		client:             client,
		once:               &sync.Once{},
		asyncPool:          asyncPool,
		prefix:             config.Redis.Prefix,
		maxLen:             config.MaxLen,
		config:             config,
		consumerHandlerDic: sync.Map{},
		failKey:            MakeLogKey(config.Redis.Prefix, "fail"),
		successKey:         MakeLogKey(config.Redis.Prefix, "success"),
	}
	var logs []ILog
	logs = append(logs, broker)
	if config.History.On {
		mongoLog := NewMongoLog(ctx, config)
		logs = append(logs, mongoLog)
	}

	broker.logJob = NewLog(asyncPool, logs...)
	broker.filter = broker
	broker.scheduleJob = broker.newScheduleJob()
	broker.captureException = defaultCaptureException

	return broker
}

func (t *RedisBroker) setCaptureException(handler func(ctx context.Context, err any)) {
	if handler != nil {
		t.captureException = handler
		t.asyncPool.captureException = handler
	}
}

func (t *RedisBroker) monitorStream(ctx context.Context, channel, topic, id string) (map[string]any, error) {

	hid := HashKey(stringx.StringToByte(id), 50)
	key := strings.Join([]string{t.prefix, channel, topic, cast.ToString(hid)}, ":")
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var (
		lastId string = "0"
		args          = redis.XReadArgs{
			Block: 5 * time.Second,
			Count: 20,
		}
		streamExist bool = false
	)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("[RedisBroker.monitor] Context error:%w", ctx.Err())
		default:

		}

		if !streamExist {
			if info := t.client.XInfoStream(ctx, key).Val(); info == nil {
				continue
			}
			streamExist = true
		}

		args.Streams = []string{key, lastId}
		cmd := t.client.XRead(ctx, &args)

		if err := cmd.Err(); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, fmt.Errorf("[RedisBroker.monitor] XRead error:%w", errors.New("pending"))
			}
			if !errors.Is(err, redis.Nil) {
				return nil, fmt.Errorf("[RedisBroker.monitor] XRead error:%w", err)
			}
			continue
		}

		messages := cmd.Val()[0].Messages
		for _, message := range messages {

			if nid, ok := message.Values["id"]; ok {
				if nid == id {
					if err := t.client.XDel(ctx, key, message.ID).Err(); err != nil {
						return nil, fmt.Errorf("[RedisBroker.monitor] XDel error:%w", err)
					}
					return message.Values, nil
				}
			}
			lastId = message.ID
		}
	}
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
		t.asyncPool.Execute(ctx, func(ctx context.Context) error {
			return t.client.ZRemRangeByScore(ctx, t.failKey, "0", cast.ToString(time.Now().UnixMilli())).Err()
		})

		// delete success logs
		t.asyncPool.Execute(ctx, func(ctx context.Context) error {
			return t.client.ZRemRangeByScore(ctx, t.successKey, "0", cast.ToString(time.Now().UnixMilli())).Err()
		})
	}
}

// Add unique id
func (t *RedisBroker) Add(ctx context.Context, key, member string) (bool, error) {
	incr := 0.001
	err := t.client.ZRank(ctx, key, member).Err()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			now := time.Now().Unix()
			return false, t.client.ZIncrBy(ctx, key, float64(now)+incr, member).Err()
		}
		return false, fmt.Errorf("[RedisBroker.Add] ZRank error:%w", err)
	}

	return true, t.client.ZIncrBy(ctx, key, incr, member).Err()
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

func (t *RedisBroker) getMessageInQueue(ctx context.Context, channel, topic string, id string) (*Message, error) {
	streamKey := MakeStreamKey(sequentialSubscribe, t.prefix, channel, topic)
	results, err := t.client.XRangeN(ctx, streamKey, "-", "+", 100).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	for _, result := range results {
		message := messageToStruct(result)
		if message.Id == id {
			return message, nil
		}
	}
	return nil, nil
}

func (t *RedisBroker) enqueue(ctx context.Context, msg *Message, dynamicOn bool) error {
	// TODO Transaction consistency should be considered here.

	// after idempotency check, before publish
	t.asyncPool.Execute(ctx, func(ctx context.Context) error {
		result := &ConsumerResult{}
		result.FillInfoByMessage(msg)
		result.Status = StatusPrepare
		return t.logJob.Archives(ctx, result)
	})

	switch msg.MoodType {
	case PUB_SUB:

		key := MakeFilter(t.prefix)
		member := msg.Id

		streamKey := MakeStreamKey(pubSubscribe, t.prefix, msg.Channel, msg.Topic)

		incr := 0.001
		if err := t.client.ZRank(ctx, key, member).Err(); err != nil {
			if errors.Is(err, redis.Nil) {
				incr = float64(time.Now().Unix()) + incr
			} else {
				return fmt.Errorf("[RedisBroker.enqueue] ZRank error:%w", err)
			}
		}
		if err := t.client.ZIncrBy(ctx, key, incr, member).Err(); err != nil {
			return fmt.Errorf("[RedisBroker.enqueue] ZIncrBy error:%w", err)
		}
		timer := timex.TimerPool.Get(15 * time.Millisecond)
		defer timex.TimerPool.Put(timer)
		var index int64 = 0
		for {
			select {
			case <-ctx.Done():
				return fmt.Errorf("[RedisBroker.enqueue] Context error:%w", ctx.Err())
			case <-timer.C:
				timer.Reset(time.Duration(rand.Int63n(15)+30) * time.Millisecond)
				if err := t.client.XAdd(ctx, redisx.NewZAddArgs(streamKey, "", "*", t.maxLen, 0, msg.ToMap())).Err(); err != nil {
					if index < 3 {
						index++
						continue
					}
					return fmt.Errorf("[RedisBroker.enqueue] XAdd error:%w", err)
				}
				return nil
			}
		}
	case SEQUENTIAL:
		streamKey := MakeStreamKey(sequentialSubscribe, t.prefix, msg.Channel, msg.Topic)

		if dynamicOn {
			err := t.client.XAdd(ctx, &redis.XAddArgs{
				Stream: "dynamic_discovery:" + msg.Channel,
				Values: map[string]interface{}{"streamKey": streamKey},
			}).Err()
			if err != nil {
				return fmt.Errorf("[RedisBroker.enqueue] seq adding dynamic key error:%w", err)
			}
		}

		xAddArgs := redisx.NewZAddArgs(streamKey, "", "*", t.maxLen, 0, msg.ToMap())
		err := t.client.XAdd(ctx, xAddArgs).Err()
		if err != nil {
			return fmt.Errorf("[RedisBroker.enqueue] seq xadd error:%w", err)
		}
	case NORMAL:
		xAddArgs := redisx.NewZAddArgs(MakeStreamKey(normalSubscribe, t.prefix, msg.Channel, msg.Topic), "", "*", t.maxLen, 0, msg.ToMap())
		err := t.client.XAdd(ctx, xAddArgs).Err()
		if err != nil {
			return fmt.Errorf("[RedisBroker.enqueue] normal xadd error:%w", err)
		}
	case DELAY:
		err := t.scheduleJob.enqueue(ctx, msg)
		if err != nil {
			return err
		}
	default:
		return errors.New("[RedisBroker.enqueue] unknown:" + msg.MoodType.String())
	}

	// publish success
	t.asyncPool.Execute(ctx, func(ctx context.Context) error {
		result := &ConsumerResult{}
		result.FillInfoByMessage(msg)
		result.Status = StatusPublished
		return t.logJob.Archives(ctx, result)
	})

	return nil
}

func (t *RedisBroker) deleteConcurrentHandler(channel string, key string) {
	consumerDicKey := MakeDynamicKey(t.prefix, channel)
	value, ok := t.consumerHandlerDic.Load(consumerDicKey)
	if !ok {
		return
	}
	handlerMap := value.(*concurrentHandlerMap)
	handlerMap.Delete(key)
}

func (t *RedisBroker) addConsumer(subType subscribeType, channel, topic string, subscribe IConsumeHandle) *RedisHandle {
	bqConfig := t.config
	handler := &RedisHandle{
		broker:              t,
		channel:             channel,
		topic:               topic,
		subscribe:           subscribe,
		subscribeType:       subType,
		deadLetterTickerDur: bqConfig.DeadLetterTicker,
		deadLetterIdleTime:  bqConfig.DeadLetterIdleTime,
		scheduleTickerDur:   defaultScheduleJobConfig.consumeTicker,
		jobMaxRetry:         bqConfig.JobMaxRetries,
		minConsumers:        bqConfig.MinConsumers,
		timeOut:             bqConfig.ConsumeTimeOut,
		wg:                  new(sync.WaitGroup),
		closeCh:             make(chan struct{}),
		resultPool: &sync.Pool{New: func() any {
			return &ConsumerResult{
				Level:   InfoLevel,
				Info:    SuccessInfo,
				RunTime: "",
			}
		}},
		errGroupPool: &sync.Pool{New: func() any {
			group := new(errgroup.Group)
			group.SetLimit(3)
			return group
		}},
		once: sync.Once{},
	}
	t.consumerHandlers = append(t.consumerHandlers, handler)
	return handler
}

func (t *RedisBroker) addDynamicConsumer(subType subscribeType, channel, topic string, subscribe IConsumeHandle, streamKey, dynamicKey string) *RedisHandle {
	bqConfig := t.config
	if dynamicKey == "" {
		dynamicKey = channel
	}
	handler := &RedisHandle{
		broker:              t,
		streamKey:           streamKey,
		dynamicKey:          dynamicKey,
		channel:             channel,
		topic:               topic,
		subscribe:           subscribe,
		subscribeType:       subType,
		deadLetterTickerDur: bqConfig.DeadLetterTicker,
		deadLetterIdleTime:  bqConfig.DeadLetterIdleTime,
		scheduleTickerDur:   defaultScheduleJobConfig.consumeTicker,
		jobMaxRetry:         bqConfig.JobMaxRetries,
		minConsumers:        bqConfig.MinConsumers,
		timeOut:             bqConfig.ConsumeTimeOut,
		wg:                  new(sync.WaitGroup),
		closeCh:             make(chan struct{}),
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
	return handler
}

func (t *RedisBroker) newScheduleJob() *scheduleJob {
	return &scheduleJob{
		broker: t,
		wg:     &sync.WaitGroup{},
		scheduleErrGroupPool: &sync.Pool{New: func() any {
			group := new(errgroup.Group)
			group.SetLimit(2)
			return group
		}},
	}
}

func (t *RedisBroker) dynamicConsuming(subType subscribeType, channel string, subscribe IConsumeHandle, dynamicKey string) {
	ctx, cancel := context.WithCancel(context.Background())
	// monitor signal
	t.waitSignal(cancel)
	t.once.Do(func() {
		err := t.logJob.Obsoletes(ctx)
		if err != nil {
			t.captureException(ctx, err)
		}

		t.asyncPool.Execute(ctx, func(ctx context.Context) error {
			t.filter.Delete(ctx, MakeFilter(t.prefix))
			return nil
		})

		logger.New().Info("Beanq dynamic consuming Start")
	})

	groupName := "read_group"
	consumerName := xid.New().String()
	discoveryStreamName := "dynamic_discovery:" + channel

	err := t.client.XGroupCreateMkStream(ctx, discoveryStreamName, groupName, "0").Err()
	if err != nil && !errors.Is(err, redis.Nil) && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		t.captureException(ctx, err)
		return
	}

	for {
		streams, err := t.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: consumerName,
			Streams:  []string{discoveryStreamName, ">"},
			Count:    1,
			Block:    0,
		}).Result()
		if err != nil {
			t.captureException(ctx, err)
			continue
		}

		consumerDicKey := MakeDynamicKey(t.prefix, channel)
		// handle
		for _, stream := range streams {
			for _, message := range stream.Messages {
				// ack
				t.client.XAck(ctx, discoveryStreamName, groupName, message.ID)
				dynamicStream := message.Values["streamKey"].(string)
				v, _ := t.consumerHandlerDic.LoadOrStore(consumerDicKey, newConcurrentHandlerMap())
				handlerMap := v.(*concurrentHandlerMap)
				if _, ok := handlerMap.Get(dynamicStream); !ok {
					channel, topic := GetChannelAndTopicFromStreamKey(dynamicStream)
					handler := t.addDynamicConsumer(subType, channel, topic, subscribe, dynamicStream, dynamicKey)
					// consume data
					if err := t.worker(ctx, handler); err != nil {
						t.captureException(ctx, err)
						continue
					}
					// REFERENCE: https://redis.io/commands/xclaim/
					// monitor other stream pending

					t.asyncPool.Execute(ctx, func(ctx context.Context) error {
						return handler.DeadLetter(ctx)
					})

					handlerMap.Set(dynamicStream, handler)
				}
			}
		}
	}
}

func (t *RedisBroker) startConsuming(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	for key, cs := range t.consumerHandlers {
		// consume data
		if err := t.worker(ctx, cs); err != nil {
			t.captureException(ctx, err)
		}

		t.asyncPool.Execute(ctx, func(ctx context.Context) error {
			cs.Schedule(ctx)
			return nil
		})

		// REFERENCE: https://redis.io/commands/xclaim/
		// monitor other stream pending
		t.asyncPool.Execute(ctx, func(ctx context.Context) error {
			return cs.DeadLetter(ctx)
		})

		t.consumerHandlers[key] = nil
	}

	t.asyncPool.Execute(ctx, func(ctx context.Context) error {
		return t.logJob.Obsoletes(ctx)
	})

	t.asyncPool.Execute(ctx, func(ctx context.Context) error {
		t.filter.Delete(ctx, MakeFilter(t.prefix))
		return nil
	})

	logger.New().Info("Beanq Start")
	// monitor signal
	<-t.waitSignal(cancel)
}

func (t *RedisBroker) worker(ctx context.Context, handle IHandle) error {
	if err := handle.Check(ctx); err != nil {
		return err
	}
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
		select {
		case <-sigs:
			_ = t.client.Close()
			t.asyncPool.Release()
			cancel()
			_ = logger.New().Sync()
			done <- true
		}
	}()
	return done
}

func (t *RedisBroker) NewMutex(name string, options ...MuxOption) *Mutex {
	pools := []redis.UniversalClient{
		t.client,
	}
	m := &Mutex{
		name:   name,
		expiry: 8 * time.Second,
		tries:  1,
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

type concurrentHandlerMap struct {
	handlerMap map[string]*RedisHandle
	mux        *sync.RWMutex
}

func newConcurrentHandlerMap() *concurrentHandlerMap {
	return &concurrentHandlerMap{handlerMap: make(map[string]*RedisHandle), mux: &sync.RWMutex{}}
}

func (h *concurrentHandlerMap) Get(key string) (*RedisHandle, bool) {
	h.mux.RLock()
	defer h.mux.RUnlock()
	value, found := h.handlerMap[key]
	return value, found
}

func (h *concurrentHandlerMap) Set(key string, value *RedisHandle) {
	h.mux.Lock()
	defer h.mux.Unlock()
	h.handlerMap[key] = value
}

func (h *concurrentHandlerMap) Delete(key string) {
	h.mux.Lock()
	defer h.mux.Unlock()
	delete(h.handlerMap, key)
}

func (h *concurrentHandlerMap) Len() int {
	h.mux.RLock()
	defer h.mux.RUnlock()
	return len(h.handlerMap)
}
