package beanq

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"github.com/retail-ai-inc/beanq/helper/stringx"
)

type RedisHandle struct {
	client           redis.UniversalClient
	log              logJobI
	consumer         DoConsumer
	deadLetterTicker *time.Ticker
	channel          string
	topic            string
	pendingIdle      time.Duration

	prefix      string
	maxLen      int64
	jobMaxRetry int
	minWorkers  int64
	timeOut     time.Duration
	pool        *ants.Pool

	wg sync.WaitGroup
}

var (
	result = sync.Pool{New: func() any {
		return &ConsumerResult{
			Level:   InfoLevel,
			Info:    SuccessInfo,
			RunTime: "",
		}
	}}

	streamArrayPool = sync.Pool{New: func() any {
		return make([]redis.XStream, 100)
	}}
)

func newRedisHandle(client redis.UniversalClient, channel, topic string, consumer DoConsumer, pool *ants.Pool) *RedisHandle {

	bqConfig := Config.Load().(BeanqConfig)
	prefix := bqConfig.Redis.Prefix
	if prefix == "" {
		prefix = DefaultOptions.Prefix
	}

	maxLen := bqConfig.Redis.MaxLen
	if maxLen <= 0 {
		maxLen = DefaultOptions.DefaultMaxLen
	}

	jobMaxRetry := bqConfig.JobMaxRetries
	if jobMaxRetry <= 0 {
		jobMaxRetry = DefaultOptions.JobMaxRetry
	}

	minWorkers := bqConfig.MinWorkers
	if minWorkers <= 0 {
		minWorkers = DefaultOptions.MinWorkers
	}
	timeOut := bqConfig.ConsumeTimeOut

	return &RedisHandle{
		client:           client,
		channel:          channel,
		topic:            topic,
		consumer:         consumer,
		log:              newLogJob(client, pool),
		deadLetterTicker: time.NewTicker(100 * time.Second),
		pendingIdle:      2 * time.Minute,
		prefix:           prefix,
		maxLen:           maxLen,
		jobMaxRetry:      jobMaxRetry,
		minWorkers:       minWorkers,
		timeOut:          timeOut,
		pool:             pool,
		wg:               sync.WaitGroup{},
	}
}

func (t *RedisHandle) Check(ctx context.Context) error {

	if err := t.checkStream(ctx); err != nil {
		return err
	}
	if err := t.checkDeadletterStream(ctx); err != nil {
		return err
	}
	return nil

}

func (t *RedisHandle) Work(ctx context.Context, done <-chan struct{}) {

	channel := t.channel
	topic := t.topic
	stream := MakeStreamKey(t.prefix, channel, topic)
	readGroupArgs := redisx.NewReadGroupArgs(channel, stream, []string{stream, ">"}, t.minWorkers, 10*time.Second)

	for {
		// check state
		select {
		case <-done:
			logger.New().Info("--------Main Task STOP--------")
			return
		case <-ctx.Done():
			logger.New().Info("--------STOP--------")
			return
		default:

		}

		// block XReadGroup to read data
		streams := t.client.XReadGroup(ctx, readGroupArgs).Val()

		if len(streams) <= 0 {
			continue
		}
		t.do(ctx, streams)
	}
}

// Please refer to http://www.redis.cn/commands/xclaim.html
func (t *RedisHandle) DeadLetter(ctx context.Context, claimDone <-chan struct{}) error {

	streamKey := MakeStreamKey(t.prefix, t.channel, t.topic)

	deadLetterStreamKey := MakeDeadLetterStreamKey(t.prefix, t.channel, t.topic)

	defer t.deadLetterTicker.Stop()

	for {
		// check state
		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				logger.New().With("", ctx.Err()).Error("context closed")
			}
			return nil
		case <-claimDone:
			logger.New().Info("--------Claim STOP--------")
			return nil
		case <-t.deadLetterTicker.C:

		}

		pendings := t.client.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: streamKey,
			Group:  t.channel,
			Start:  "-",
			End:    "+",
			Count:  100,
		}).Val()

		if len(pendings) <= 0 {
			continue
		}

		for _, pending := range pendings {

			if pending.Idle < t.pendingIdle {
				continue
			}

			// if pending retry count > 10,then add it into dead_letter_stream
			if pending.RetryCount > 10 {
				val := t.client.XRangeN(ctx, streamKey, pending.ID, "+", 1).Val()
				if len(val) <= 0 {
					continue
				}

				msg := Message(val[0])
				msg.Values["pendingRetry"] = pending.RetryCount

				xAddArgs := redisx.NewZAddArgs(deadLetterStreamKey, "", "*", t.maxLen, 0, msg.Values)
				if err := t.client.XAdd(ctx, xAddArgs).Err(); err != nil {
					logger.New().Error(err)
				}
				if err := t.client.XDel(ctx, streamKey, pending.ID).Err(); err != nil {
					logger.New().Error(err)
				}
			} else {
				if err := t.client.XClaim(ctx, &redis.XClaimArgs{
					Stream:   streamKey,
					Group:    t.channel,
					Consumer: pending.Consumer,
					MinIdle:  t.pendingIdle,
					Messages: []string{pending.ID},
				}).Err(); err != nil {
					logger.New().Error(err)
				}
			}

		}
		continue
	}
}

func (t *RedisHandle) do(ctx context.Context, streams []redis.XStream) {

	channel := t.channel
	for key, v := range streams {

		stream := v.Stream
		message := v.Messages

		t.wg.Add(len(v.Messages))
		for _, vv := range message {
			nv := vv
			if err := t.pool.Submit(func() {
				msg := Message(nv)

				r, err := t.execute(ctx, &msg)
				if err != nil {
					r.Level = ErrLevel
					r.Info = FlagInfo(err.Error())
				}

				// Successfully consumed data, stored in `string`
				if err := t.log.saveLog(ctx, r); err != nil {
					logger.New().Error(err)
				}

				r = &ConsumerResult{Level: InfoLevel, Info: SuccessInfo, RunTime: ""}
				result.Put(r)

				if err := t.ack(ctx, stream, channel, nv.ID); err != nil {
					logger.New().Error(err)
				}
				t.wg.Done()
			}); err != nil {
				logger.New().Error(err)
			}

		}
		streams[key] = redis.XStream{}
	}
	t.wg.Wait()
}

func (t *RedisHandle) ack(ctx context.Context, stream, channel string, ids ...string) error {

	// `stream` confirmation message
	err := t.client.XAck(ctx, stream, channel, ids...).Err()
	// delete data from `stream`
	err = t.client.XDel(ctx, stream, ids...).Err()
	return err

}

type BeanqContext struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

var BeanqCtx = sync.Pool{New: func() any {
	timeOut := Config.Load().(BeanqConfig).ConsumeTimeOut
	ctx, cancel := context.WithTimeout(context.Background(), timeOut)
	return BeanqContext{Ctx: ctx, Cancel: cancel}
}}

func (t *RedisHandle) execute(ctx context.Context, msg *Message) (*ConsumerResult, error) {

	r := result.Get().(*ConsumerResult)
	r.Id = msg.Id()
	r.BeginTime = time.Now()
	// if error,then retry to consume
	nerr := make(chan error, 1)

	retryCount, err := RetryInfo(ctx, func() error {
		nctx := BeanqCtx.Get().(BeanqContext)

		defer func() {
			nctx.Cancel()
			BeanqCtx.Put(nctx)

			if ne := recover(); ne != nil {
				nerr <- fmt.Errorf("error:%+v,stack:%s", ne, stringx.ByteToString(debug.Stack()))
			}
		}()

		err := make(chan error, 1)
		if oterr := t.pool.Submit(func() {
			err <- t.consumer(nctx.Ctx, msg)
		}); oterr != nil {
			return oterr
		}

		select {
		case <-nctx.Ctx.Done():
			return nctx.Ctx.Err()
		case ferr := <-err:
			return ferr
		}
	}, t.jobMaxRetry)
	if err != nil {
		nerr <- err
	}

	select {
	case v := <-nerr:
		return r, v
	default:

	}
	r.EndTime = time.Now()

	sub := r.EndTime.Sub(r.BeginTime)

	r.AddTime = msg.AddTime()

	r.Retry = retryCount
	r.Payload = msg.Payload()
	r.RunTime = sub.String()
	r.ExecuteTime = msg.ExecuteTime()
	r.Topic = msg.Topic()
	r.Channel = t.channel

	return r, nil
}

// checkStream   if stream not exist,then create it
func (t *RedisHandle) checkStream(ctx context.Context) error {

	normalStreamKey := MakeStreamKey(t.prefix, t.channel, t.topic)
	return t.check(ctx, normalStreamKey)

}

func (t *RedisHandle) checkDeadletterStream(ctx context.Context) error {

	// if dead letter stream don't exist,then create it
	deadLetterStreamKey := MakeDeadLetterStreamKey(t.prefix, t.channel, t.topic)
	return t.check(ctx, deadLetterStreamKey)

}

func (t *RedisHandle) check(ctx context.Context, streamName string) error {
	result := t.client.XInfoGroups(ctx, streamName).Val()
	if len(result) < 1 {
		if err := t.client.XGroupCreateMkStream(ctx, streamName, t.channel, "0").Err(); err != nil {
			return err
		}
	}
	return nil
}
