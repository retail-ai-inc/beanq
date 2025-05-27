package bredis

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	public "github.com/retail-ai-inc/beanq/v3/internal"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/retail-ai-inc/beanq/v3/internal/capture"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

type RdbBroker struct {
	client         redis.UniversalClient
	prefix         string
	maxLen         int64
	consumers      int64
	deadLetterIdle time.Duration
}

func NewBroker(client redis.UniversalClient, prefix string, maxLen, consumers int64, duration time.Duration) *RdbBroker {
	return &RdbBroker{
		client:         client,
		prefix:         prefix,
		maxLen:         maxLen,
		consumers:      consumers,
		deadLetterIdle: duration,
	}
}

func (t *RdbBroker) Mood(moodType btype.MoodType, config *capture.Config) public.IBroker {
	if moodType == btype.NORMAL {
		return NewNormal(t.client, t.prefix, t.maxLen, t.consumers, t.deadLetterIdle, config)
	}
	if moodType == btype.SEQUENTIAL {
		return NewSequential(t.client, t.prefix, t.consumers, t.deadLetterIdle, config)
	}
	if moodType == btype.DELAY {
		return NewSchedule(t.client, t.prefix, t.consumers, t.deadLetterIdle, config)
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
		consumers      int64
		captureConfig  *capture.Config
	}
)

var DefaultBlockDuration BlockDuration = func() time.Duration {
	return time.Duration(rand.Int63n(9)+1) * time.Second
}

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

	for range ticker.C {

		select {
		case <-ctx.Done():
			_ = t.client.Close()
			return
		default:
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
			//Replace `Del` with `Unlink` and hand it over to the Redis server for processing.
			if err := t.client.Unlink(ctx, deadLetterKey).Err(); err != nil {
				capture.Dlq.When(t.captureConfig).If(&capture.Channel{
					Channel: channel,
					Topic:   []string{},
				}).Then(err)
				logger.New().Error(err)
			}
			continue
		}

		pending := pendings[0]

		if pending.Idle > deadLetterIdleTime {

			rangeV := t.client.XRange(ctx, streamKey, pending.ID, pending.ID).Val()

			if len(rangeV) <= 0 {
				t.client.Unlink(ctx, deadLetterKey)
				continue
			}
			val := rangeV[0].Values
			val["logType"] = bstatus.Dlq

			// logic XAddArgs
			args := &redis.XAddArgs{
				Stream: logicKey,
				Values: val,
			}

			if v, ok := val["status"]; ok {
				if v.(string) == bstatus.StatusPublished {
					//  TODO:Re-enter the queue
					var maxLenInt int64 = 2000
					if maxLen, ok := val["maxLen"]; ok {
						maxLenInt = maxLen.(int64)
					}
					// normal message XAddArgs
					// Need to handle idempotent keys
					args = NewZAddArgs(streamKey, "", "*", maxLenInt, 0, val)
				}
			}
			if err := t.client.XAdd(ctx, args).Err(); err != nil {
				capture.Dlq.When(t.captureConfig).If(&capture.Channel{Channel: channel, Topic: []string{}}).Then(err)
				logger.New().Error(err)
			}
		}

		if _, err := t.client.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
			pipeliner.XAck(ctx, streamKey, channel, pending.ID)
			pipeliner.XDel(ctx, streamKey, pending.ID)
			return nil
		}); err != nil {
			capture.Dlq.When(t.captureConfig).If(&capture.Channel{Channel: channel, Topic: []string{}}).Then(err)
			logger.New().Error(err)
		}
		if err := t.client.Unlink(ctx, deadLetterKey).Err(); err != nil {
			capture.Dlq.When(t.captureConfig).If(&capture.Channel{Channel: channel, Topic: []string{}}).Then(err)
			logger.New().Error(err)
		}
	}
}

func (t *Base) Enqueue(_ context.Context, _ map[string]any) error {
	return nil
}

func (t *Base) Dequeue(ctx context.Context, channel, topic string, do public.CallBack) {
	streamKey := tool.MakeStreamKey(t.subType, t.prefix, channel, topic)
	readGroupArgs := NewReadGroupArgs(channel, streamKey, []string{streamKey, ">"}, t.consumers, 500*time.Millisecond)
	// worker num
	workerNum := runtime.GOMAXPROCS(0) - 1
	if workerNum <= 0 {
		workerNum = 2
	}

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
		}

		streams := cmd.Val()
		if len(streams) <= 0 {
			continue
		}
		stream := streams[0].Stream
		messages := streams[0].Messages

		// Use Fan-Out mode to process tasks
		var wait sync.WaitGroup
		jobs, results := make(chan public.Stream, len(messages)), make(chan public.Stream, len(messages))
		// start workers
		// open core(num-1) goroutines
		for i := 0; i < workerNum; i++ {
			wait.Add(1)
			go worker(ctx, jobs, results, do, &wait)
		}
		// send jobs
		for _, message := range messages {
			jobs <- public.Stream{
				Data:    message.Values,
				Id:      message.ID,
				Channel: channel,
				Stream:  stream,
			}
		}
		close(jobs)
		go func() {
			wait.Wait()
			close(results)
		}()
		// handler worker results
		ids := make([]string, 0, len(messages))
		for result := range results {
			if err := t.AddLog(ctx, result.Data); err != nil {
				logger.New().Error(err)
				capture.Fail.When(t.captureConfig).If(&capture.Channel{Channel: channel, Topic: []string{topic}}).Then(err)
				continue
			}
			ids = append(ids, result.Id)
		}
		_, err := t.client.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
			pipeliner.XAck(ctx, stream, channel, ids...)
			pipeliner.XDel(ctx, stream, ids...)
			return nil
		})
		if err != nil {
			logger.New().Error(err)
			capture.Fail.When(t.captureConfig).If(&capture.Channel{Channel: channel, Topic: []string{topic}}).Then(err)
		}
	}
}

// consumer worker
func worker(ctx context.Context, jobs, result chan public.Stream, handler public.CallBack, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {

		select {
		case <-ctx.Done():
			return
		default:

		}
		val := job.Data

		val["status"] = bstatus.StatusReceived
		val["beginTime"] = time.Now()
		timeToRun := cast.ToDuration(val["timeToRun"])
		sessionCtx, cancel := context.WithTimeout(context.Background(), timeToRun)

		retry, err := tool.RetryInfo(sessionCtx, func() (err error) {
			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("[panic recover]: %+v\n%s\n", p, debug.Stack())
				}
			}()
			err = handler(sessionCtx, val)

			return
		}, cast.ToInt(val["retry"]))

		if err != nil {
			if h, ok := interface{}(handler).(interface {
				Error(ctx context.Context, err error)
			}); ok {
				h.Error(sessionCtx, err)
			}
			val["level"] = bstatus.ErrLevel
			val["info"] = err.Error()
			val["status"] = bstatus.StatusFailed
		} else {
			val["status"] = bstatus.StatusSuccess
		}

		val["endTime"] = time.Now()
		val["retry"] = retry
		val["runTime"] = cast.ToTime(val["endTime"]).Sub(cast.ToTime(val["beginTime"])).String()
		hostname, _ := os.Hostname()
		val["hostName"] = hostname
		// `stream` confirmation message
		cancel()
		job.Data = val
		result <- job
	}
}
