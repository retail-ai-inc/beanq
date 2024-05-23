package beanq

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"golang.org/x/sync/errgroup"
)

type RedisHandle struct {
	broker             *RedisBroker
	subscribe          IConsumeHandle
	deadLetterTicker   *time.Ticker
	channel            string
	topic              string
	deadLetterIdleTime time.Duration
	subscribeType      subscribeType
	// errorCallbacks   []ErrorCallback

	jobMaxRetry  int
	minConsumers int64
	timeOut      time.Duration

	wg           *sync.WaitGroup
	resultPool   *sync.Pool
	errGroupPool *sync.Pool
	once         sync.Once
}

func (t *RedisHandle) Check(ctx context.Context) error {
	if err := t.checkStream(ctx); err != nil {
		return err
	}
	return nil
}

func (t *RedisHandle) Channel() string {
	return t.channel
}

func (t *RedisHandle) Topic() string {
	return t.topic
}

func (t *RedisHandle) Process(ctx context.Context) {

	switch t.subscribeType {
	case normalSubscribe:
		t.runSubscribe(ctx)
	case sequentialSubscribe:
		t.runSequentialSubscribe(ctx)
	}

}

func (t *RedisHandle) runSubscribe(ctx context.Context) {
	channel := t.channel
	topic := t.topic
	stream := MakeStreamKey(t.subscribeType, t.broker.prefix, channel, topic)
	readGroupArgs := redisx.NewReadGroupArgs(channel, stream, []string{stream, ">"}, t.minConsumers, 10*time.Second)

	for {
		// check state
		select {
		case <-ctx.Done():
			logger.New().Info("Main Task Stop")
			return
		default:

		}

		// block XReadGroup to read data
		streams := t.broker.client.XReadGroup(ctx, readGroupArgs).Val()

		if len(streams) <= 0 {
			continue
		}
		t.do(ctx, streams)
	}
}

func (t *RedisHandle) runSequentialSubscribe(ctx context.Context) {
	stream := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)

	readGroupArgs := redisx.NewReadGroupArgs(t.channel, stream, []string{stream, ">"}, 1, 10*time.Second)

	mutex := t.broker.NewMutex(
		strings.Join([]string{t.broker.prefix, t.channel, t.topic, "seq_sync"}, ":"),
		WithExpiry(20*time.Second),
	)

	for {
		select {
		case <-ctx.Done():
			logger.New().Info("Sequential Task Stop")
			return

		case <-time.After(time.Millisecond * 200):
			err := t.broker.client.Watch(ctx, func(tx *redis.Tx) error {
				xp, err := tx.XPending(ctx, stream, readGroupArgs.Group).Result()
				if err != nil {
					return err
				}

				streamInfo := tx.XInfoStream(ctx, stream).Val()
				if streamInfo == nil || (streamInfo.Length-xp.Count) <= 0 {
					return errors.New("queue data is empty")
				}
				return nil
			}, stream)

			if err != nil {
				continue
			}

			if err := mutex.LockContext(ctx); err != nil {
				logger.New().Error(err)
				continue
			}

			cmd := t.broker.client.XReadGroup(ctx, readGroupArgs)
			vals := cmd.Val()
			if len(vals) <= 0 {
				if _, err := mutex.UnlockContext(ctx); err != nil {
					logger.New().Error(err)
				}
				continue
			}

			stream := vals[0].Stream

			for _, v := range vals[0].Messages {
				nv := v
				message := messageToStruct(nv.Values)

				result := t.resultPool.Get().(*ConsumerResult).FillInfoByMessage(message)

				group := t.errGroupPool.Get().(*errgroup.Group)

				result.Status = StatusExecuting
				result.BeginTime = time.Now()
				nctx, cancel := context.WithTimeout(context.Background(), message.TimeToRun)

				retry, err := RetryInfo(nctx, func() error {
					if err := t.subscribe.Handle(nctx, message); err != nil {
						if h, ok := t.subscribe.(IConsumeCancel); ok {
							return h.Cancel(nctx, message)
						}
					}
					return nil
				}, message.Retry)
				if err != nil {
					if h, ok := t.subscribe.(IConsumeError); ok {
						h.Error(nctx, err)
					}
					result.Level = ErrLevel
					result.Info = FlagInfo(err.Error())
					result.Status = StatusFailed
				} else {
					result.Status = StatusSuccess
				}

				result.EndTime = time.Now()
				result.Retry = retry
				result.RunTime = result.EndTime.Sub(result.BeginTime).String()

				cancel()
				group.TryGo(func() error {
					// `stream` confirmation message
					if err := t.broker.client.XAck(ctx, stream, t.channel, nv.ID).Err(); err != nil {
						return err
					}
					// delete data from `stream`
					if err := t.broker.client.XDel(ctx, stream, nv.ID).Err(); err != nil {
						return err
					}
					// set result for ack
					err = t.broker.client.SetNX(ctx, strings.Join([]string{t.broker.prefix, t.channel, t.topic, "status", result.Id}, ":"), result, time.Hour).Err()
					if err != nil {
						return err
					}
					return nil
				})

				group.TryGo(func() error {
					defer t.resultPool.Put(result)
					return t.broker.logJob.Archives(ctx, result)

				})

				if err := group.Wait(); err != nil {
					logger.New().Error(err)
				}

				t.errGroupPool.Put(group)
			}

			if _, err := mutex.UnlockContext(ctx); err != nil {
				logger.New().Error(err)
			}
		}
	}
}

// DeadLetter Please refer to https://redis.io/docs/latest/commands/xclaim/
func (t *RedisHandle) DeadLetter(ctx context.Context) error {
	streamKey := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)
	defer t.deadLetterTicker.Stop()

	for {
		// check state
		select {
		case <-ctx.Done():
			logger.New().Info("DeadLetter Work Stop")
			return nil
		case <-t.deadLetterTicker.C:

		}

		pendings := t.broker.client.XPendingExt(ctx, &redis.XPendingExtArgs{
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
			// if pending idle  > pending duration(20 * time.Minute),then add it into dead_letter_stream
			if pending.Idle > t.deadLetterIdleTime {
				val := t.broker.client.XRangeN(ctx, streamKey, pending.ID, "+", 1).Val()
				if len(val) <= 0 {
					// the message is not in stream, but in the pending list. need to ack it.
					log.Printf("Message ID %s not found in the stream, removing from pending\n", pending.ID)
					t.broker.client.XAck(ctx, streamKey, t.channel, pending.ID)
					continue
				}

				msg := messageToStruct(val[0])
				// msg.Values["pendingRetry"] = pending.RetryCount
				// msg.Values["idle"] = pending.Idle.Seconds()

				r := t.resultPool.Get().(*ConsumerResult).FillInfoByMessage(msg)
				r.EndTime = time.Now()
				r.Retry = msg.Retry

				r.RunTime = r.EndTime.Sub(r.BeginTime).String()
				r.Level = ErrLevel
				r.Info = "too long pending"

				if err := t.broker.logJob.Archives(ctx, r); err != nil {
					logger.New().Error(err)
				}

				if err := t.broker.client.XDel(ctx, streamKey, val[0].ID).Err(); err != nil {
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
			if err := t.broker.pool.Submit(func() {
				r := t.execute(ctx, &nv)

				if err := t.ack(ctx, stream, channel, nv.ID); err != nil {
					logger.New().Error(err)
					return
				}
				if err := t.broker.logJob.Archives(ctx, r); err != nil {
					logger.New().Error(err)
					return
				}

				defer t.wg.Done()
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
	err := t.broker.client.XAck(ctx, stream, channel, ids...).Err()
	// delete data from `stream`
	err = t.broker.client.XDel(ctx, stream, ids...).Err()
	return err

}

func (t *RedisHandle) execute(ctx context.Context, message *redis.XMessage) *ConsumerResult {
	msg := messageToStruct(message)
	r := t.resultPool.Get().(*ConsumerResult).FillInfoByMessage(msg)

	nctx, cancel := context.WithTimeout(context.Background(), msg.TimeToRun)

	defer func() {
		r = &ConsumerResult{Level: InfoLevel, Info: SuccessInfo, RunTime: ""}
		t.resultPool.Put(r)
		cancel()
	}()

	r.Status = StatusExecuting
	r.BeginTime = time.Now()
	retryCount, err := RetryInfo(ctx, func() error {
		return t.subscribe.Handle(nctx, msg)
	}, msg.Retry)

	r.EndTime = time.Now()
	r.Retry = retryCount
	r.RunTime = r.EndTime.Sub(r.BeginTime).String()

	if err != nil {
		if h, ok := t.subscribe.(IConsumeError); ok {
			h.Error(nctx, err)
		}
		r.Level = ErrLevel
		r.Info = FlagInfo(err.Error())
	}
	return r
}

// checkStream   if stream not exist,then create it
func (t *RedisHandle) checkStream(ctx context.Context) error {

	normalStreamKey := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)
	return t.check(ctx, normalStreamKey)

}

func (t *RedisHandle) checkDeadletterStream(ctx context.Context) error {

	// if dead letter stream don't exist,then create it
	deadLetterStreamKey := MakeDeadLetterStreamKey(t.broker.prefix, t.channel, t.topic)
	return t.check(ctx, deadLetterStreamKey)

}

func (t *RedisHandle) check(ctx context.Context, streamName string) error {
	result := t.broker.client.XInfoGroups(ctx, streamName).Val()
	if len(result) < 1 {
		if err := t.broker.client.XGroupCreateMkStream(ctx, streamName, t.channel, "0").Err(); err != nil {
			return err
		}
	}
	return nil
}
