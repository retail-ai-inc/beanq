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

func SwitchBroker(client redis.UniversalClient, prefix string, maxLen int64, moodType btype.MoodType) public.IBroker {

	if moodType == btype.NORMAL {
		return NewNormal(client, prefix, maxLen)
	}
	if moodType == btype.SEQUENTIAL {
		return NewSequential(client, prefix)
	}
	if moodType == btype.DELAY {
		return NewSchedule(client, prefix)
	}
	return nil
}

type (
	BlockDuration func() time.Duration
	Base          struct {
		client redis.UniversalClient
		public.IProcessLog
		prefix        string
		subType       btype.SubscribeType
		blockDuration BlockDuration
		errGroup      sync.Pool
	}
)

var (
	DefaultBlockDuration BlockDuration = func() time.Duration {
		return time.Duration(rand.Int63n(9)+1) * time.Second
	}
)

func (t *Base) DeadLetter(ctx context.Context, channel, topic string) {

	streamKey := tool.MakeStreamKey(t.subType, t.prefix, channel, topic)
	deadLetterKey := tool.MakeDeadLetterKey(t.prefix)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	r := t.errGroup.Get().(*errgroup.Group)
	defer func() {
		t.errGroup.Put(r)
	}()

	var deadLetterIdleTime = 20 * time.Second

	for {
		// check state
		select {

		case <-ctx.Done():
			_ = t.client.Close()
			return
		case <-ticker.C:

		}
		//t.client.XPending()
		pendings := t.client.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream:   streamKey,
			Group:    channel,
			Consumer: streamKey,
			Start:    "-",
			End:      "+",
			Count:    100,
		}).Val()

		if len(pendings) <= 0 {
			continue
		}
		ids := make([]string, 0, len(pendings))
		readStreams := make([][]string, 0, len(pendings))
		for _, pending := range pendings {
			// if pending idle  > pending duration(20 * time.Minute),then add it into dead_letter_stream
			if pending.Idle > deadLetterIdleTime {
				ids = append(ids, pending.ID)
				readStreams = append(readStreams, []string{streamKey, pending.ID})
			}

		}
		fmt.Printf("stream:%+v \n", readStreams)
		//t.client.XGroupCreateMkStream(ctx, deadLetterKey, tool.BeanqLogGroup, "0")

		//t.client.XGroupCreateMkStream(ctx, deadLetterKey, tool.BeanqDeadLetterGroup, "0")

		//v1 := t.client.XRead(ctx, &redis.XReadArgs{
		//	Streams: readStreams,
		//	Count:   int64(len(readStreams)),
		//}).Val()
		//fmt.Printf("数据：%+v \n", v1)

		//cmd := t.client.XClaim(ctx, &redis.XClaimArgs{
		//	Stream:   deadLetterKey,
		//	Group:    tool.BeanqDeadLetterGroup,
		//	Consumer: tool.BeanqDeadLetterConsumer,
		//	Messages: ids,
		//	MinIdle:  10 * time.Second,
		//})
		//result, _ := cmd.Result()
		//fmt.Printf("result:%+v \n", result)
		//fmt.Printf("cmd:%+v \n", cmd.String())
		//fmt.Printf("%+v \n", pendings)

		//v := t.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		//	Group:    tool.BeanqDeadLetterGroup,
		//	Consumer: tool.BeanqDeadLetterConsumer,
		//	Streams:  []string{deadLetterKey},
		//	Count:    10,
		//}).Val()
		//fmt.Printf("vvvvv:%+v \n", v)

		watcher := func(tx *redis.Tx) error {
			_, err := tx.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
				//t.client.XClaim(ctx, &redis.XClaimArgs{
				//	Stream:   deadLetterKey,
				//	Group:    tool.BeanqLogGroup,
				//	Consumer: tool.BeanqDeadLetterConsumer,
				//	Messages: ids,
				//})
				//t.client.XAck(ctx, streamKey, channel, ids...)
				//t.client.XDel(ctx, streamKey, ids...)
				return nil
			})
			return err
		}

		if err := t.client.Watch(ctx, watcher, deadLetterKey, streamKey); err != nil {
			fmt.Println(err)
		}
		continue
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

	//id := stream.Id
	//stm := stream.Stream
	//channel := stream.Channel

	var val map[string]any
	val = stream.Data

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
			//pipeliner.XAck(ctx, stm, channel, id)
			//pipeliner.XDel(ctx, stm, id)
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
