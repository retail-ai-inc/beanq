package bredis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
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
)

type RdbBroker struct {
	client           redis.UniversalClient
	prefix           string
	maxLen           int64
	consumers        int64
	consumerPoolSize int
	deadLetterIdle   time.Duration
}

func NewBroker(client redis.UniversalClient, prefix string, maxLen, consumers int64, consumerPoolSize int, duration time.Duration) *RdbBroker {
	return &RdbBroker{
		client:           client,
		prefix:           prefix,
		maxLen:           maxLen,
		consumers:        consumers,
		consumerPoolSize: consumerPoolSize,
		deadLetterIdle:   duration,
	}
}

func (t *RdbBroker) Mood(moodType btype.MoodType, config *capture.Config) public.IBroker {
	if moodType == btype.NORMAL {
		return NewNormal(t.client, t.prefix, t.maxLen, t.consumers, t.consumerPoolSize, t.deadLetterIdle, config)
	}
	if moodType == btype.SEQUENCE {
		return NewSequence(t.client, t.prefix, t.consumers, t.consumerPoolSize, t.deadLetterIdle, config)
	}
	if moodType == btype.DELAY {
		return NewSchedule(t.client, t.prefix, t.consumers, t.consumerPoolSize, t.deadLetterIdle, config)
	}
	if moodType == btype.SEQUENCE_BY_LOCK {
		return NewSequenceByLock(t.client, t.prefix, t.consumers, t.consumerPoolSize, t.deadLetterIdle, config)
	}
	return nil
}

type (
	BlockDuration func() time.Duration
	Base          struct {
		client redis.UniversalClient
		public.IProcessLog
		blockDuration    BlockDuration
		prefix           string
		subType          btype.SubscribeType
		deadLetterIdle   time.Duration
		consumers        int64
		consumerPoolSize int
		captureConfig    *capture.Config
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
	workerNum := t.consumerPoolSize

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
		for i := 0; i < workerNum; i++ {
			wait.Add(1)
			go worker(ctx, jobs, results, do, &wait, t.captureConfig)
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

			if orderKey, ok := result.Data["orderKey"]; ok {
				orderRediKey := tool.MakeSequenceLockKey(t.prefix, channel, topic, cast.ToString(orderKey))
				t.client.HDel(ctx, orderRediKey)
			}

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
func worker(ctx context.Context, jobs, result chan public.Stream, handler public.CallBack, wg *sync.WaitGroup, config *capture.Config) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		return
	case job, ok := <-jobs:
		if !ok {
			return
		}

		val := job.Data
		//deep copy for handler: prevent data race.
		//In the future, maybe only `payload`,`channel`,`topic` will be needed
		copiedVal := make(map[string]any, len(val))
		for k, v := range val {
			copiedVal[k] = v
		}

		now := time.Now()
		val["status"] = bstatus.StatusReceived
		val["beginTime"] = now

		var timeToRunLimit []time.Duration
		if err := json.Unmarshal([]byte((val["timeToRunLimit"]).(string)), &timeToRunLimit); err != nil {
			capture.Fail.When(config).If(&capture.Channel{Channel: job.Channel, Topic: []string{job.Stream}}).Then(err)
			return
		}
		timeToRunLimitLen := len(timeToRunLimit)

		timeToRun := cast.ToDuration(val["timeToRun"])
		sessionCtx, cancel := context.WithTimeout(context.Background(), timeToRun)

		retry, err := tool.RetryInfo(sessionCtx, func() (handlerErr error) {
			defer func() {
				if p := recover(); p != nil {
					handlerErr = fmt.Errorf("[panic recover]: %+v\n%s\n", p, debug.Stack())
				}
			}()
			if timeToRunLimitLen > 0 {
				go func(limit []time.Duration) {
					ticker := time.NewTicker(time.Second)
					defer ticker.Stop()
					i := 0

					for {
						select {
						case <-sessionCtx.Done():
							return
						case <-ticker.C:
							if i >= timeToRunLimitLen {
								return
							}
							if time.Since(now) >= limit[i] {
								i++
								capErr := fmt.Errorf("Info:Task execution timeout,Body:%+v \n", copiedVal)
								capture.System.When(config).If(nil).Then(capErr)
							}
						}
					}
				}(timeToRunLimit)
			}

			handlerErr = handler(sessionCtx, copiedVal)

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
