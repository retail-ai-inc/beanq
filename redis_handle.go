package beanq

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"golang.org/x/sync/errgroup"
)

type RedisHandle struct {
	broker           *RedisBroker
	subscribe        IConsumeHandle
	deadLetterTicker *time.Ticker
	channel          string
	topic            string
	pendingIdle      time.Duration
	subscribeType    subscribeType
	// errorCallbacks   []ErrorCallback

	jobMaxRetry  int
	minConsumers int64
	timeOut      time.Duration

	wg           *sync.WaitGroup
	result       *sync.Pool
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

func (t *RedisHandle) Process(ctx context.Context, done, seqDone <-chan struct{}) {
	switch t.subscribeType {
	case normalSubscribe:
		t.runSubscribe(ctx, done)
	case sequentialSubscribe:
		t.runSequentialSubscribe(ctx, seqDone)
	}
}

func (t *RedisHandle) runSubscribe(ctx context.Context, done <-chan struct{}) {
	channel := t.channel
	topic := t.topic
	stream := MakeStreamKey(t.subscribeType, t.broker.prefix, channel, topic)
	readGroupArgs := redisx.NewReadGroupArgs(channel, stream, []string{stream, ">"}, t.minConsumers, 10*time.Second)

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
		streams := t.broker.client.XReadGroup(ctx, readGroupArgs).Val()

		if len(streams) <= 0 {
			continue
		}
		t.do(ctx, streams)
	}
}

func (t *RedisHandle) runSequentialSubscribe(ctx context.Context, done <-chan struct{}) {
	stream := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)

	key := strings.Join([]string{t.broker.prefix, t.channel, t.topic, "seq_id"}, ":")

	readGroupArgs := redisx.NewReadGroupArgs(t.channel, stream, []string{stream, ">"}, 1, 10*time.Second)

	// timer := time.NewTimer(time.Millisecond * 100)

	result := t.result.Get().(*ConsumerResult)

	group := t.errGroupPool.Get().(*errgroup.Group)

	keyExDuration := 20 * time.Second

	defer func() {
		// timer.Stop()
		result = &ConsumerResult{Level: InfoLevel, Info: SuccessInfo, RunTime: ""}
	}()

	for {
		select {
		case <-done:
			logger.New().Info("--------Sequential Task STOP--------")
			return
		case <-ctx.Done():
			return
		case <-time.After(time.Millisecond * 100):
			var executable bool
			err := t.broker.client.Watch(ctx, func(tx *redis.Tx) error {
				executingStatus := tx.Get(ctx, key).Val()
				streamInfo := tx.XInfoStream(ctx, stream).Val()

				_, err := tx.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
					if (streamInfo == nil || streamInfo.Length == 0) && executingStatus != "" {
						err := pipeliner.SetEX(ctx, key, "", keyExDuration).Err()
						return err
					}

					if executingStatus == "executing" {
						return nil
					}

					if err := pipeliner.SetEX(ctx, key, "executing", keyExDuration).Err(); err != nil {
						return err
					}
					executable = true
					return nil
				})
				return err
			}, key, stream)

			if err != nil {
				if !errors.Is(err, redis.TxFailedErr) {
					logger.New().Error(err)
				}
				continue
			}

			if !executable {
				continue
			}

			cmd := t.broker.client.XReadGroup(ctx, readGroupArgs)
			vals := cmd.Val()
			if len(vals) <= 0 {
				continue
			}

			stream := vals[0].Stream

			for _, v := range vals[0].Messages {
				nv := v
				message := messageToStruct(nv.Values)

				result.Id = message.Id
				result.BeginTime = time.Now()
				nctx, cancel := context.WithTimeout(context.Background(), message.TimeToRun)

				retry, err := RetryInfo(nctx, func() error {
					if err := t.subscribe.Handle(nctx, message); err != nil {
						if h, ok := t.subscribe.(IConsumeCancel); ok {
							return h.Cancel(nctx, message)
						}
					}
					return nil
				}, t.jobMaxRetry)

				result.EndTime = time.Now()
				sub := result.EndTime.Sub(result.BeginTime)
				result.AddTime = message.AddTime
				result.Retry = retry
				result.Payload = message.Payload
				result.Priority = message.Priority
				result.RunTime = sub.String()
				result.ExecuteTime = message.ExecuteTime
				result.Topic = message.Topic
				result.Channel = t.channel
				result.MoodType = message.MoodType
				if err != nil {
					if h, ok := t.subscribe.(IConsumeError); ok {
						h.Error(nctx, err)
					}
					result.Level = ErrLevel
					result.Info = FlagInfo(err.Error())
				}

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
					return nil
				})
				group.TryGo(func() error {
					return t.broker.logJob.saveLog(ctx, result)
				})
				if err := group.Wait(); err != nil {
					t.broker.client.SetEX(ctx, key, "", keyExDuration)
					logger.New().Error(err)
				}
				t.errGroupPool.Put(group)

			}
			t.broker.client.SetEX(ctx, key, "", keyExDuration)
		}
	}
}

// DeadLetter Please refer to http://www.redis.cn/commands/xclaim.html
func (t *RedisHandle) DeadLetter(ctx context.Context, claimDone <-chan struct{}) error {
	streamKey := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)
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
			if pending.Idle < t.pendingIdle {
				continue
			}
			// if pending retry count > 5,then add it into dead_letter_stream
			if pending.RetryCount > 5 {
				val := t.broker.client.XRangeN(ctx, streamKey, pending.ID, "+", 1).Val()
				if len(val) <= 0 {
					continue
				}

				msg := messageToStruct(val[0])
				// msg.Values["pendingRetry"] = pending.RetryCount
				// msg.Values["idle"] = pending.Idle.Seconds()

				r := t.result.Get().(*ConsumerResult)
				r.Id = msg.Id
				r.BeginTime = msg.ExecuteTime

				r.EndTime = time.Now()
				sub := r.EndTime.Sub(r.BeginTime)
				r.AddTime = msg.AddTime
				r.Retry = msg.Retry
				r.Payload = msg.Payload
				r.RunTime = sub.String()
				r.ExecuteTime = msg.ExecuteTime
				r.Topic = msg.Topic
				r.Channel = t.channel
				r.MoodType = msg.MoodType

				r.Level = ErrLevel
				r.Info = "too long pending"

				if err := t.broker.logJob.saveLog(ctx, r); err != nil {
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

				group := t.errGroupPool.Get().(*errgroup.Group)
				group.TryGo(func() error {
					return t.ack(ctx, stream, channel, nv.ID)
				})
				group.TryGo(func() error {
					return t.broker.logJob.saveLog(ctx, r)
				})
				if err := group.Wait(); err != nil {
					logger.New().Error(err)
				}
				t.errGroupPool.Put(group)

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
	r := t.result.Get().(*ConsumerResult)
	// var cancel context.CancelFunc
	msg := messageToStruct(message)

	nctx, cancel := context.WithTimeout(context.Background(), msg.TimeToRun)

	defer func() {
		r = &ConsumerResult{Level: InfoLevel, Info: SuccessInfo, RunTime: ""}
		t.result.Put(r)
		cancel()
	}()

	r.Id = msg.Id
	r.BeginTime = time.Now()

	retryCount, err := RetryInfo(ctx, func() error {
		return t.subscribe.Handle(nctx, msg)
	}, t.jobMaxRetry)

	r.EndTime = time.Now()
	sub := r.EndTime.Sub(r.BeginTime)
	r.AddTime = msg.AddTime
	r.Retry = retryCount
	r.Payload = msg.Payload
	r.Priority = msg.Priority
	r.RunTime = sub.String()
	r.ExecuteTime = msg.ExecuteTime
	r.Topic = msg.Topic
	r.Channel = t.channel
	r.MoodType = msg.MoodType

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
