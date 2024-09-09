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

func (t *RedisHandle) runSubscribe(ctx context.Context) {
	channel := t.channel
	topic := t.topic
	streamKey := MakeStreamKey(t.subscribeType, t.broker.prefix, channel, topic)
	readGroupArgs := redisx.NewReadGroupArgs(channel, streamKey, []string{streamKey, ">"}, t.minConsumers, 10*time.Second)

	for {
		cmd := t.broker.client.XReadGroup(ctx, readGroupArgs)
		if err := cmd.Err(); err != nil {

			if strings.Contains(err.Error(), "NOGROUP No such key") {
				if err := t.broker.client.XGroupCreateMkStream(ctx, streamKey, channel, "0").Err(); err != nil {
					logger.New().Error(err)
					return
				}
				continue
			}

			if errors.Is(err, context.Canceled) || errors.Is(err, redis.ErrClosed) {
				logger.New().Info("Channel:[", t.channel, "]Topic:[", t.topic, "] Main Task Stop")
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
				result := messageToStruct(msg)
				group := rh.errGroupPool.Get().(*errgroup.Group)
				defer func() {
					wait.Done()
					rh.errGroupPool.Put(group)
					rh.resultPool.Put(result)
				}()

				nmessage := messageToStruct(msg.Values)

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
					result.Info = err.Error()
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
	readGroupArgs := redisx.NewReadGroupArgs(t.channel, streamKey, []string{streamKey, ">"}, 20, 10*time.Second)

	for {
		cmd := t.broker.client.XReadGroup(ctx, readGroupArgs)
		if err := cmd.Err(); err != nil {

			if strings.Contains(err.Error(), "NOGROUP No such key") {
				if err := t.broker.client.XGroupCreateMkStream(ctx, streamKey, t.channel, "0").Err(); err != nil {
					logger.New().Error(err)
					return
				}
				continue
			}

			if errors.Is(err, context.Canceled) || errors.Is(err, redis.ErrClosed) {
				logger.New().Info("Channel:[", t.channel, "]Topic:[", t.topic, "] Sequential Task Stop")
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

				group := rh.errGroupPool.Get().(*errgroup.Group)

				result := messageToStruct(vv)

				defer func() {
					if p := recover(); p != nil {
						// receive the message
						clone := result
						rh.broker.asyncPool.Execute(ctx, func(ctx context.Context) error {
							clone.Status = StatusFailed
							clone.Info = fmt.Sprintf("[panic recover]: %+v\n%s\n", p, debug.Stack())
							return rh.broker.Archive(ctx, clone)
						})
						rh.broker.captureException(ctx, p)
					}
					wait.Done()
					rh.errGroupPool.Put(group)
				}()

				result.Status = StatusReceived
				result.BeginTime = time.Now()

				sessionCtx, cancel := context.WithTimeout(context.Background(), result.TimeToRun)
				retry, err := RetryInfo(sessionCtx, func() error {
					var globalErr error
					if err := rh.subscribe.Handle(sessionCtx, result); err != nil {
						if errors.Is(err, NilHandle) {
							globalErr = errors.Join(globalErr, nil)
						} else {
							globalErr = errors.Join(globalErr, err)
							if h, ok := rh.subscribe.(IConsumeCancel); ok {
								if err := h.Cancel(sessionCtx, result); err != nil {

									if errors.Is(err, NilCancel) {
										globalErr = errors.Join(globalErr, nil)
									} else {
										globalErr = errors.Join(globalErr, err)
									}

								}
							}
						}
					}
					return globalErr
				}, result.Retry)

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

				client := rh.broker.client
				group.TryGo(func() error {
					_, err := client.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
						pipeliner.XAck(ctx, stream, rh.channel, id)
						pipeliner.XDel(ctx, stream, id)
						return nil
					})
					return err
				})
				group.TryGo(func() error {
					if err := rh.broker.Archive(ctx, result); err != nil {
						return err
					}
					return nil
				})
				if err := group.Wait(); err != nil {
					rh.broker.captureException(ctx, err)
					return
				}

			}(msg.ID, msg.Values, t)
		}
		wait.Wait()
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
			logger.New().Info("Channel:[", t.channel, "]Topic:[", t.topic, "] DeadLetter Work Stop")
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

				r := messageToStruct(val[0])

				// r.FillInfoByMessage(msg)
				r.EndTime = time.Now()
				// r.Retry = msg.Retry
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
