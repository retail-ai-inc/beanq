package bredis

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/retail-ai-inc/beanq/v3/internal"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
	"math/rand"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

func SwitchBroker(client redis.UniversalClient, prefix string, maxLen int64, deadLetterIdle time.Duration, moodType btype.MoodType) public.IBroker {

	if moodType == btype.NORMAL {
		return NewNormal(client, prefix, maxLen, deadLetterIdle)
	}
	if moodType == btype.SEQUENTIAL {
		return NewSequential(client, prefix, deadLetterIdle)
	}
	if moodType == btype.DELAY {
		return NewSchedule(client, prefix, deadLetterIdle)
	}
	return nil
}

type (
	BlockDuration func() time.Duration
	Base          struct {
		errGroup sync.Pool
		client   redis.UniversalClient
		public.IProcessLog
		blockDuration  BlockDuration
		prefix         string
		subType        btype.SubscribeType
		deadLetterIdle time.Duration
	}
)

var (
	DefaultBlockDuration BlockDuration = func() time.Duration {
		return time.Duration(rand.Int63n(9)+1) * time.Second
	}
)

func (t *Base) DeadLetter(ctx context.Context, channel, topic string) {

	streamKey := tool.MakeStreamKey(t.subType, t.prefix, channel, topic)
	logicKey := tool.MakeLogicKey(t.prefix)
	deadLetterKey := strings.Join([]string{streamKey, "dead_letter_lock"}, ":")

	ticker := time.NewTicker(DefaultBlockDuration())
	defer ticker.Stop()

	r := t.errGroup.Get().(*errgroup.Group)
	defer func() {
		t.errGroup.Put(r)
	}()

	deadLetterIdleTime := t.deadLetterIdle

	for {
		select {

		case <-ctx.Done():
			_ = t.client.Close()
			return
		case <-ticker.C:

		}
		if v := AddLogicLockScript.Run(ctx, t.client, []string{deadLetterKey}).Val(); v.(int64) == 1 {
			continue
		}

		pendings := t.client.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream:   streamKey,
			Group:    channel,
			Consumer: streamKey,
			Start:    "-",
			End:      "+",
			Count:    1,
		}).Val()
		length := len(pendings)
		if length <= 0 {
			if err := t.client.Del(ctx, deadLetterKey).Err(); err != nil {
				logger.New().Error(err)
			}
			continue
		}

		watcher := func(tx *redis.Tx) error {

			pending := pendings[0]
			_, err := tx.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {

				if pending.Idle > deadLetterIdleTime {

					rangeV := t.client.XRange(ctx, streamKey, pending.ID, pending.ID).Val()
					if len(rangeV) <= 0 {
						return nil
					}
					val := rangeV[0].Values
					val["type"] = "dead_letter"
					t.client.XAdd(ctx, &redis.XAddArgs{
						Stream: logicKey,
						Values: val,
					})
				}
				t.client.XAck(ctx, streamKey, channel, pending.ID)
				t.client.XDel(ctx, streamKey, pending.ID)
				return nil
			})
			return err
		}

		if err := t.client.Watch(ctx, watcher, streamKey, logicKey); err != nil {
			logger.New().Error(err)
		}
		if err := t.client.Del(ctx, deadLetterKey).Err(); err != nil {
			logger.New().Error(err)
		}
	}
}

func (t *Base) Enqueue(ctx context.Context, data map[string]any) error {
	return nil
}

func (t *Base) Dequeue(ctx context.Context, channel, topic string, do public.CallBack) {

	streamKey := tool.MakeStreamKey(t.subType, t.prefix, channel, topic)

	readGroupArgs := NewReadGroupArgs(channel, streamKey, []string{streamKey, ">"}, 10, t.blockDuration())

	for {

		cmd := t.client.XReadGroup(ctx, readGroupArgs)
		if err := cmd.Err(); err != nil {

			if strings.Contains(err.Error(), "NOGROUP No such key") {
				if err := t.client.XGroupCreateMkStream(ctx, streamKey, channel, "0").Err(); err != nil {
					logger.New().Error(err)
					return
				}
				continue
			}

			if errors.Is(err, context.Canceled) || errors.Is(err, redis.ErrClosed) {
				_ = t.client.Close()
				logger.New().Info("Channel:[", channel, "]Topic:[", topic, "] Task Stop")
				return
			}
			continue
		}

		streams := cmd.Val()
		stream := streams[0].Stream
		messages := streams[0].Messages

		var wait sync.WaitGroup
		for _, message := range messages {
			wait.Add(1)
			go func(msg redis.XMessage) {
				vv := msg.Values
				defer func() {
					if p := recover(); p != nil {
						// receive the message
						vv["status"] = bstatus.StatusFailed
						vv["info"] = fmt.Sprintf("[panic recover]: %+v\n%s\n", p, debug.Stack())
						if err := t.AddLog(ctx, vv); err != nil {
							logger.New().Error(err)
						}
					}
					wait.Done()
				}()
				if err := t.Consumer(ctx, &public.Stream{
					Data:    vv,
					Id:      msg.ID,
					Channel: channel,
					Stream:  stream,
				}, do); err != nil {
					logger.New().Error(err)
				}
			}(message)
		}
		wait.Wait()
	}
}

func (t *Base) Consumer(ctx context.Context, stream *public.Stream, handler public.CallBack) error {

	id := stream.Id
	stm := stream.Stream
	channel := stream.Channel

	val := stream.Data

	group := t.errGroup.Get().(*errgroup.Group)
	defer func() {
		t.errGroup.Put(group)
	}()
	val["status"] = bstatus.StatusReceived
	val["beginTime"] = time.Now()
	timeToRun := cast.ToDuration(val["timeToRun"])
	sessionCtx, cancel := context.WithTimeout(context.Background(), timeToRun)

	retry, err := tool.RetryInfo(sessionCtx, func() error {
		return handler(sessionCtx, val)
	}, 3)

	if err != nil {
		//if h, ok := rh.subscribe.(IConsumeError); ok {
		//	h.Error(sessionCtx, err)
		//}
		val["level"] = bstatus.ErrLevel
		val["info"] = err.Error()
		val["status"] = bstatus.StatusFailed
	} else {
		val["status"] = bstatus.StatusSuccess
	}

	val["endTime"] = time.Now()
	val["retry"] = retry
	val["runTime"] = cast.ToTime(val["endTime"]).Sub(cast.ToTime(val["beginTime"])).String()
	// `stream` confirmation message
	cancel()
	// ------------------------
	group.TryGo(func() error {
		_, err := t.client.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
			pipeliner.XAck(ctx, stm, channel, id)
			pipeliner.XDel(ctx, stm, id)
			return nil
		})
		return err
	})
	group.TryGo(func() error {
		return t.AddLog(ctx, val)
	})
	if err := group.Wait(); err != nil {
		return err
	}
	return nil
}
