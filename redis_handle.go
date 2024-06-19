package beanq

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

type RedisHandle struct {
	broker        *RedisBroker
	channel       string
	topic         string
	subscribeType subscribeType
	subscribe     IConsumeHandle

	deadLetterTicker   *time.Ticker
	deadLetterIdleTime time.Duration

	scheduleTicker *time.Ticker

	jobMaxRetry  int
	minConsumers int64
	timeOut      time.Duration

	wg           *sync.WaitGroup
	resultPool   *sync.Pool
	errGroupPool *sync.Pool
	once         sync.Once
	closeCh      chan struct{}
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
		case <-t.closeCh:
			return
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

func (t *RedisHandle) Schedule(ctx context.Context) {
	// timeWheel To be implemented
	defer t.scheduleTicker.Stop()

	var (
		now      time.Time
		timeUnit        = MakeTimeUnit(t.broker.prefix, t.channel, t.topic)
		scoreMin string = "0"
		scoreMax string
	)
	for {
		select {
		case <-t.closeCh:
			return
		case <-ctx.Done():
			t.broker.pool.Release()
			logger.New().Info("Schedule Task Stop")
			return

		case <-t.scheduleTicker.C:
		}

		now = time.Now()

		scoreMax = cast.ToString(now.UnixMilli() + 1)
		err := t.broker.client.Watch(ctx, func(tx *redis.Tx) error {
			val, err := tx.ZRangeByScore(ctx, timeUnit, &redis.ZRangeBy{
				Min:    scoreMin,
				Max:    scoreMax,
				Offset: 0,
				Count:  1,
			}).Result()

			if err != nil {
				return err
			}

			if len(val) <= 0 {
				scoreMin = scoreMax
			} else {
				scoreMin = val[0]
				if err := tx.ZRem(ctx, timeUnit, val[0]).Err(); err != nil {
					return err
				}
			}
			return nil
		}, timeUnit)
		if err != nil {
			logger.New().Error(err)
			continue
		}

		if err := t.broker.scheduleJob.doConsume(ctx, scoreMax, t.channel, t.topic); err != nil {
			logger.New().With("", err).Error("consume err")
			// continue
		}
	}
}

func (t *RedisHandle) runSequentialSubscribe(ctx context.Context) {
	stream := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)

	readGroupArgs := redisx.NewReadGroupArgs(t.channel, stream, []string{stream, ">"}, 100, 10*time.Second)

	subKey := MakeSubKey(t.broker.prefix, t.channel, t.topic)

	duration := time.Millisecond * 100
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		timer.Reset(duration)
		select {
		case <-t.closeCh:
			return
		case <-ctx.Done():
			logger.New().Info("Sequential Task Stop")
			return

		case <-timer.C:
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

			cmd := t.broker.client.XReadGroup(ctx, readGroupArgs)
			vals := cmd.Val()
			if len(vals) <= 0 {
				continue
			}

			stream := vals[0].Stream
			for _, v := range vals[0].Messages {
				message := messageToStruct(v.Values)
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
						return err
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

				// t.broker.client.Del(ctx, strings.Join([]string{t.broker.prefix, message.Channel, message.Topic, message.Id}, ":"))
				group.TryGo(func() error {
					// `stream` confirmation message
					if err := t.broker.client.XAck(ctx, stream, t.channel, v.ID).Err(); err != nil {
						return err
					}
					// delete data from `stream`
					if err := t.broker.client.XDel(ctx, stream, v.ID).Err(); err != nil {
						return err
					}
					return nil
				})

				group.TryGo(func() error {
					// set result for ack
					var m map[string]any
					data, err := result.MarshalBinary()
					if err != nil {
						return err
					}
					if err := json.Unmarshal(data, &m); err != nil {
						return err
					}

					if err := t.broker.client.XAdd(ctx, &redis.XAddArgs{
						Stream: subKey + "_" + m["Id"].(string),
						Values: m,
					}).Err(); err != nil {
						return err
					}
					return nil
				})

				group.TryGo(func() error {
					// fix data race
					clone := *result
					return t.broker.logJob.Archives(ctx, &clone)
				})

				if err := group.Wait(); err != nil {
					logger.New().Error(err)
				}
				t.errGroupPool.Put(group)
				t.resultPool.Put(result)
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
		case <-t.closeCh:
			return nil
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
				t.resultPool.Put(r)
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

func (t *RedisHandle) checkDeadLetterStream(ctx context.Context) error {

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

func (t *RedisHandle) close() error {
	// safe close
	select {
	case <-t.closeCh:
		// already close
		return nil
	default:
		close(t.closeCh)
	}

	return nil
}
