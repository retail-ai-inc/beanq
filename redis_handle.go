package beanq

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

type RedisHandle struct {
	broker        *RedisBroker
	streamKey     string
	dynamicKey    string
	channel       string
	topic         string
	subscribeType subscribeType
	subscribe     IConsumeHandle

	deadLetterTickerDur time.Duration
	deadLetterIdleTime  time.Duration

	scheduleTickerDur time.Duration

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
		t.runSeqSubscribe(ctx)
	}
}

// nolint: unused
func (t *RedisHandle) pubSubscribe(ctx context.Context) {
	defer t.close()

	streamKey := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)
	readGroupArgs := redisx.NewReadGroupArgs(t.channel, streamKey, []string{streamKey, ">"}, t.minConsumers, 5*time.Second)

	for {

		cmd := t.broker.client.XReadGroup(ctx, readGroupArgs)
		if err := cmd.Err(); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.New().Info("Pub/Sub Task Stop")
				return
			}
			if !errors.Is(err, redis.Nil) {
				logger.New().Error(err)
				var randNum int64 = rand.Int63n(50) + 50
				time.Sleep(time.Duration(randNum) * time.Millisecond)
			}
			continue
		}

		streams := cmd.Val()
		stream := streams[0].Stream
		messages := streams[0].Messages

		var wait sync.WaitGroup
		for _, msg := range messages {

			wait.Add(1)
			go func(vv redis.XMessage, rh *RedisHandle) {

				result := rh.resultPool.Get().(*ConsumerResult)
				group := rh.errGroupPool.Get().(*errgroup.Group)
				defer func() {
					wait.Done()
					rh.errGroupPool.Put(group)
					rh.resultPool.Put(result)
				}()

				message := messageToStruct(vv.Values)

				result.FillInfoByMessage(message)
				result.Status = StatusReceived
				result.BeginTime = time.Now()
				sessionCtx, cancel := context.WithTimeout(context.Background(), message.TimeToRun)

				retry, err := RetryInfo(sessionCtx, func() error {
					if err := rh.subscribe.Handle(sessionCtx, message); err != nil {
						if h, ok := rh.subscribe.(IConsumeCancel); ok {
							return h.Cancel(sessionCtx, message)
						}
						return err
					}
					return nil
				}, message.Retry)

				if err != nil {
					if h, ok := rh.subscribe.(IConsumeError); ok {
						h.Error(sessionCtx, err)
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
				// ------------------------
				client := rh.broker.client

				group.TryGo(func() error {
					if err := rh.broker.logJob.Archives(ctx, *result); err != nil {
						return fmt.Errorf("log error:%w", err)
					}
					return nil
				})
				val := vv.Values
				group.TryGo(func() error {
					id := HashKey([]byte(message.Id), 50)
					val["status"] = result.Status
					streamkey := strings.Join([]string{rh.broker.prefix, rh.channel, rh.topic, cast.ToString(id)}, ":")
					return client.XAdd(ctx, redisx.NewZAddArgs(streamkey, "", "*", rh.broker.maxLen, 0, val)).Err()
				})
				group.TryGo(func() error {
					_, err := client.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
						pipeliner.XAck(ctx, stream, t.channel, vv.ID)
						pipeliner.XDel(ctx, stream, vv.ID)
						return nil
					})
					return err
				})

				if err := group.Wait(); err != nil {
					t.broker.captureException(ctx, err)
					return
				}
			}(msg, t)
		}
		wait.Wait()
	}
}

func (t *RedisHandle) runSubscribe(ctx context.Context) {
	channel := t.channel
	topic := t.topic
	streamKey := MakeStreamKey(t.subscribeType, t.broker.prefix, channel, topic)
	readGroupArgs := redisx.NewReadGroupArgs(channel, streamKey, []string{streamKey, ">"}, t.minConsumers, 10*time.Second)

	for {
		cmd := t.broker.client.XReadGroup(ctx, readGroupArgs)
		if err := cmd.Err(); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, redis.ErrClosed) {
				logger.New().Info("Channel:", t.channel, " Topic:", t.topic, " Main Task Stop")
				return
			}
			if !errors.Is(err, redis.Nil) {
				logger.New().Error(err)
				var randNum int64 = rand.Int63n(50) + 50
				time.Sleep(time.Duration(randNum) * time.Millisecond)
			}
			continue
		}

		streams := cmd.Val()
		stream := streams[0].Stream
		messages := streams[0].Messages

		var wait sync.WaitGroup
		for _, message := range messages {
			wait.Add(1)
			go func(msg redis.XMessage, rh *RedisHandle) {
				result := rh.resultPool.Get().(*ConsumerResult)
				group := rh.errGroupPool.Get().(*errgroup.Group)
				defer func() {
					wait.Done()
					rh.errGroupPool.Put(group)
					rh.resultPool.Put(result)
				}()

				nmessage := messageToStruct(msg.Values)

				result.FillInfoByMessage(nmessage)
				result.Status = StatusReceived
				result.BeginTime = time.Now()
				sessionCtx, cancel := context.WithTimeout(context.Background(), nmessage.TimeToRun)

				retry, err := RetryInfo(sessionCtx, func() error {
					if err := rh.subscribe.Handle(sessionCtx, nmessage); err != nil {
						if h, ok := rh.subscribe.(IConsumeCancel); ok {
							return h.Cancel(sessionCtx, nmessage)
						}
						return err
					}
					return nil
				}, nmessage.Retry)

				if err != nil {
					if h, ok := rh.subscribe.(IConsumeError); ok {
						h.Error(sessionCtx, err)
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
				// `stream` confirmation message

				cancel()
				// ------------------------
				group.TryGo(func() error {
					if err := rh.broker.client.XAck(ctx, stream, t.channel, msg.ID).Err(); err != nil {
						return err
					}
					if err := rh.broker.client.XDel(ctx, stream, msg.ID).Err(); err != nil {
						return err
					}
					return nil
				})
				group.TryGo(func() error {
					return rh.broker.logJob.Archives(ctx, *result)
				})
				if err := group.Wait(); err != nil {
					logger.New().Error(err)
					return
				}
			}(message, t)
		}
		wait.Wait()
	}
}

func (t *RedisHandle) Schedule(ctx context.Context) error {
	err := t.broker.scheduleJob.run(ctx, t.channel, t.topic, t.closeCh)
	if err != nil {
		return fmt.Errorf("[RedisHandle.Schedule] run error: %w", err)
	}
	return nil
}

func (t *RedisHandle) runSeqSubscribe(ctx context.Context) {
	defer t.close()

	streamKey := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)
	readGroupArgs := redisx.NewReadGroupArgs(t.channel, streamKey, []string{streamKey, ">"}, 20, 1*time.Minute)

	for {
		cmd := t.broker.client.XReadGroup(ctx, readGroupArgs)
		if err := cmd.Err(); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, redis.ErrClosed) {
				logger.New().Info("Channel:", t.channel, " Topic:", t.topic, " Pub/Sub Task Stop")
				return
			}
			if !errors.Is(err, redis.Nil) {
				t.broker.captureException(ctx, err)
			}
			continue
		}

		streams := cmd.Val()
		if len(streams) <= 0 {
			continue
		}

		stream := streams[0].Stream
		messages := streams[0].Messages

		var wait sync.WaitGroup
		for _, msg := range messages {
			msg := msg
			wait.Add(1)
			go func(id string, vv map[string]any, rh *RedisHandle) {
				result1 := rh.resultPool.Get().(*ConsumerResult)
				group := rh.errGroupPool.Get().(*errgroup.Group)

				message := messageToStruct(vv)
				result := result1.FillInfoByMessage(message)

				defer func() {
					if p := recover(); p != nil {
						// receive the message
						clone := *result
						rh.broker.asyncPool.Execute(ctx, func(ctx context.Context) error {
							clone.Status = StatusFailed
							clone.Info = fmt.Sprintf("[panic recover]: %+v\n%s\n", p, debug.Stack())
							return rh.broker.logJob.Archives(ctx, clone)
						})
						rh.broker.captureException(ctx, p)
					}
					wait.Done()
					rh.errGroupPool.Put(group)
					rh.resultPool.Put(result)
				}()

				ch := make(chan ConsumerResult, 2)
				// Collaborative synchronization of data to prevent data pollution
				go func(nch chan ConsumerResult) {
					index := 0
					for {
						if index == 2 {
							return
						}
						select {
						case v, ok := <-nch:
							if ok {
								index++
								if err := rh.broker.logJob.Archives(ctx, v); err != nil {
									rh.broker.captureException(ctx, err)
								}
							}
						}
					}
				}(ch)

				result.Status = StatusReceived
				result.BeginTime = time.Now()

				// receive the message
				ch <- *result

				sessionCtx, cancel := context.WithTimeout(context.Background(), message.TimeToRun)
				retry, err := RetryInfo(sessionCtx, func() error {
					if err := rh.subscribe.Handle(sessionCtx, message); err != nil {
						if h, ok := rh.subscribe.(IConsumeCancel); ok {
							return h.Cancel(sessionCtx, message)
						}
						return err
					}
					return nil
				}, message.Retry)

				result.Status = StatusSuccess
				if err != nil {
					if h, ok := rh.subscribe.(IConsumeError); ok {
						h.Error(sessionCtx, err)
					}
					result.Level = ErrLevel
					result.Info = err.Error()
					result.Status = StatusFailed
				}

				result.EndTime = time.Now()
				result.Retry = retry
				result.RunTime = result.EndTime.Sub(result.BeginTime).String()

				cancel()
				// ------------------------
				client := rh.broker.client
				group.TryGo(func() error {
					_, err := client.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
						pipeliner.XAck(ctx, stream, rh.channel, id)
						pipeliner.XDel(ctx, stream, id)
						return nil
					})
					return err
				})
				// result of execute
				ch <- *result

				if err := group.Wait(); err != nil {
					rh.broker.captureException(ctx, err)
					return
				}

			}(msg.ID, msg.Values, t)
		}
		wait.Wait()
	}
}

func (t *RedisHandle) runSequentialSubscribe(ctx context.Context) {
	defer t.close()

	stream := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)

	readGroupArgs := redisx.NewReadGroupArgs(t.channel, stream, []string{stream, ">"}, 1000, 0)

	// dynamicKey only affects the mutex name.
	// If the dynamicKey and topic of different channels are the same, there will be a lock.
	mutex := t.broker.NewMutex(
		strings.Join([]string{t.broker.prefix, t.dynamicKey, t.topic, "seq_sync"}, ":"),
		WithExpiry(20*time.Second),
	)

	result := t.resultPool.Get().(*ConsumerResult)
	group := t.errGroupPool.Get().(*errgroup.Group)
	defer func() {
		t.errGroupPool.Put(group)
		t.resultPool.Put(result)
	}()

	duration := time.Millisecond * 100
	timer := time.NewTimer(duration)
	defer timer.Stop()

	deadline := time.Second * 5
	deadlineTimer := time.NewTimer(deadline)
	defer deadlineTimer.Stop()

	streamKey := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)
	deadLetterTicker := time.NewTicker(t.deadLetterTickerDur)
	defer deadLetterTicker.Stop()
	r := t.resultPool.Get().(*ConsumerResult)
	defer func() {
		t.resultPool.Put(r)
	}()

	for {
		timer.Reset(duration)
		select {
		case <-ctx.Done():
			logger.New().Info("Sequential Task Stop")
			return
		case <-deadlineTimer.C:
			// No new message before deadline
			return
		case <-timer.C:
			count, err := t.broker.client.XLen(ctx, stream).Result()
			if err != nil {
				t.broker.captureException(ctx, err)
				continue
			}
			if count == 0 {
				continue
			}

			// If there is new messages, reset the deadline.
			deadlineTimer.Reset(deadline)

			func() {
				if err := mutex.LockContext(ctx); err != nil {
					return
				}
				defer func() {
					if _, err := mutex.UnlockContext(ctx); err != nil {
						t.broker.captureException(ctx, err)
					}
				}()

				cmd := t.broker.client.XReadGroup(ctx, readGroupArgs)
				vals := cmd.Val()
				if len(vals) <= 0 {
					return
				}
				for _, v := range vals {
					stream := v.Stream
					for _, vv := range v.Messages {
						message := messageToStruct(vv.Values)

						result.FillInfoByMessage(message)
						result.Status = StatusReceived
						result.BeginTime = time.Now()

						t.broker.asyncPool.Execute(ctx, func(ctx context.Context) error {
							return t.broker.logJob.Archives(ctx, *result)
						})

						sessionCtx, cancel := context.WithTimeout(context.Background(), message.TimeToRun)

						retry, err := RetryInfo(sessionCtx, func() error {
							if err := t.subscribe.Handle(sessionCtx, message); err != nil {
								if h, ok := t.subscribe.(IConsumeCancel); ok {
									return h.Cancel(sessionCtx, message)
								}
								return err
							}
							return nil
						}, message.Retry)

						if err != nil {
							if h, ok := t.subscribe.(IConsumeError); ok {
								h.Error(sessionCtx, err)
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
						// `stream` confirmation message
						if err = t.broker.client.XAck(ctx, stream, t.channel, vv.ID).Err(); err != nil {
							t.broker.captureException(ctx, err)
						}

						t.broker.client.XAdd(ctx, &redis.XAddArgs{
							Stream:     strings.Join([]string{t.broker.prefix, t.channel, t.topic, message.Id}, ":"),
							NoMkStream: false,
							MaxLen:     0,
							Approx:     false,
							Limit:      0,
							Values:     vv.Values,
						})

						cancel()
						group.TryGo(func() error {
							// delete data from `stream`
							if err := t.broker.client.XDel(ctx, stream, vv.ID).Err(); err != nil {
								return err
							}
							return nil
						})

						// fix data race
						group.TryGo(func() error {
							return t.broker.logJob.Archives(ctx, *result)
						})

						if err := group.Wait(); err != nil {
							t.broker.captureException(ctx, err)
						}

					}
				}
			}()
		case <-deadLetterTicker.C:
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
					r.FillInfoByMessage(msg)
					r.EndTime = time.Now()
					r.Retry = msg.Retry
					r.Status = StatusDeadLetter
					r.RunTime = r.EndTime.Sub(r.BeginTime).String()
					r.Level = ErrLevel
					r.Info = "too long pending"

					if err := t.broker.logJob.Archives(ctx, *r); err != nil {
						t.broker.captureException(ctx, err)
					}

					if err := t.broker.client.XDel(ctx, streamKey, val[0].ID).Err(); err != nil {
						t.broker.captureException(ctx, err)
					}
				}
			}
		}
	}
}

// DeadLetter Please refer to https://redis.io/docs/latest/commands/xclaim/
func (t *RedisHandle) DeadLetter(ctx context.Context) error {
	streamKey := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)
	ticker := time.NewTicker(t.deadLetterTickerDur)
	defer ticker.Stop()
	r := t.resultPool.Get().(*ConsumerResult)
	defer func() {
		t.resultPool.Put(r)
	}()
	for {
		// check state
		select {
		case <-t.closeCh:
			return nil
		case <-ctx.Done():
			logger.New().Info("DeadLetter Work Stop")
			return nil
		case <-ticker.C:

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

				r.FillInfoByMessage(msg)
				r.EndTime = time.Now()
				r.Retry = msg.Retry
				r.Status = StatusDeadLetter
				r.RunTime = r.EndTime.Sub(r.BeginTime).String()
				r.Level = ErrLevel
				r.Info = "too long pending"

				if err := t.broker.logJob.Archives(ctx, *r); err != nil {
					t.broker.captureException(ctx, err)
				}

				if err := t.broker.client.XDel(ctx, streamKey, val[0].ID).Err(); err != nil {
					t.broker.captureException(ctx, err)
				}
			}
		}
		continue
	}
}

// checkStream   if stream not exist,then create it
func (t *RedisHandle) checkStream(ctx context.Context) error {
	normalStreamKey := MakeStreamKey(t.subscribeType, t.broker.prefix, t.channel, t.topic)
	return t.check(ctx, normalStreamKey)
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
	case _, ok := <-t.closeCh:
		if !ok {
			// already closed
			return nil
		}
	default:
		close(t.closeCh)
		if t.streamKey != "" {
			t.broker.deleteConcurrentHandler(t.channel, t.streamKey)
		}
	}

	return nil
}
