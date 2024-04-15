package beanq

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"golang.org/x/sync/errgroup"
)

type RedisHandle struct {
	client           redis.UniversalClient
	log              ILogJob
	run              any
	deadLetterTicker *time.Ticker
	channel          string
	topic            string
	pendingIdle      time.Duration

	prefix       string
	maxLen       int64
	jobMaxRetry  int
	minConsumers int64
	timeOut      time.Duration
	pool         *ants.Pool

	wg                  *sync.WaitGroup
	result              *sync.Pool
	errGroupPool        *sync.Pool
	once                sync.Once
	normalDone, seqDone chan struct{}
}

func newRedisHandle(client redis.UniversalClient, channel, topic string, run any, pool *ants.Pool) *RedisHandle {

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

	minConsumers := bqConfig.MinConsumers
	if minConsumers <= 0 {
		minConsumers = DefaultOptions.MinConsumers
	}
	timeOut := bqConfig.ConsumeTimeOut

	return &RedisHandle{
		client:           client,
		channel:          channel,
		topic:            topic,
		run:              run,
		log:              newLogJob(client, pool),
		deadLetterTicker: time.NewTicker(100 * time.Second),
		pendingIdle:      2 * time.Minute,
		prefix:           prefix,
		maxLen:           maxLen,
		jobMaxRetry:      jobMaxRetry,
		minConsumers:     minConsumers,
		timeOut:          timeOut,
		pool:             pool,
		wg:               new(sync.WaitGroup),
		result: &sync.Pool{New: func() any {
			return &ConsumerResult{
				Level:   InfoLevel,
				Info:    SuccessInfo,
				RunTime: "",
			}
		}},
		errGroupPool: &sync.Pool{New: func() any {
			group := new(errgroup.Group)
			group.SetLimit(2)
			return group
		}},
		once:       sync.Once{},
		normalDone: make(chan struct{}, 1),
		seqDone:    make(chan struct{}, 1),
	}
}

func (t *RedisHandle) Check(ctx context.Context) error {

	if err := t.checkStream(ctx); err != nil {
		return err
	}
	return nil

}

func (t *RedisHandle) RunSubscribe(ctx context.Context, done <-chan struct{}) {

	channel := t.channel
	topic := t.topic
	stream := MakeStreamKey(t.prefix, channel, topic)
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
		streams := t.client.XReadGroup(ctx, readGroupArgs).Val()

		if len(streams) <= 0 {
			continue
		}
		t.do(ctx, streams)
	}
}

func (t *RedisHandle) RunSequentialSubscribe(ctx context.Context, done <-chan struct{}) {

	stream := MakeStreamKey(t.prefix, t.channel, t.topic)
	key := strings.Join([]string{t.prefix, t.channel, t.topic, "seq_id"}, ":")

	readGroupArgs := redisx.NewReadGroupArgs(t.channel, stream, []string{stream, ">"}, 1, 10*time.Second)

	ticker := time.NewTicker(time.Second)

	result := t.result.Get().(*ConsumerResult)

	group := t.errGroupPool.Get().(*errgroup.Group)

	keyExDuration := 20 * time.Second

	nctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

	defer func() {
		ticker.Stop()
		cancel()
		result = &ConsumerResult{Level: InfoLevel, Info: SuccessInfo, RunTime: ""}
	}()
	for {
		select {
		case <-done:
			logger.New().Info("--------Sequential Task STOP--------")
			return
		case <-ctx.Done():
			return
		case <-ticker.C:

			err := t.client.Watch(ctx, func(tx *redis.Tx) error {
				if tx.Get(ctx, key).Val() == "" {
					if err := tx.SetEX(ctx, key, 1, keyExDuration).Err(); err != nil {
						return err
					}
				}
				return nil
			}, key)
			if err != nil {
				continue
			}
			cmd := t.client.XReadGroup(ctx, readGroupArgs)
			vals := cmd.Val()
			if len(vals) <= 0 {
				t.client.SetEX(ctx, key, "", keyExDuration)
				continue
			}

			stream := vals[0].Stream
			for _, v := range vals[0].Messages {
				nv := v
				message := messageToStruct(nv.Values)

				result.Id = message.Id
				result.BeginTime = time.Now()

				retry, err := RetryInfo(nctx, func() error {

					if err := t.run.(ISequentialConsumer).Run(message); err != nil {
						if err := t.run.(ISequentialConsumer).Cancel(message); err != nil {
							return err
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
				result.Topic = message.TopicName
				result.Channel = t.channel
				result.MoodType = message.MoodType
				if err != nil {
					t.run.(ISequentialConsumer).Error(err)
					result.Level = ErrLevel
					result.Info = FlagInfo(err.Error())
				}

				group.TryGo(func() error {
					// `stream` confirmation message
					if err := t.client.XAck(ctx, stream, t.channel, nv.ID).Err(); err != nil {
						return err
					}
					// delete data from `stream`
					if err := t.client.XDel(ctx, stream, nv.ID).Err(); err != nil {
						return err
					}
					return nil
				})
				group.TryGo(func() error {
					return t.log.saveLog(ctx, result)
				})
				if err := group.Wait(); err != nil {
					t.client.SetEX(ctx, key, "", keyExDuration)
					logger.New().Error(err)
				}
				t.errGroupPool.Put(group)

			}
			t.client.SetEX(ctx, key, "", keyExDuration)
		}
	}
}

// Please refer to http://www.redis.cn/commands/xclaim.html
func (t *RedisHandle) DeadLetter(ctx context.Context, claimDone <-chan struct{}) error {

	streamKey := MakeStreamKey(t.prefix, t.channel, t.topic)

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
			// if pending retry count > 5,then add it into dead_letter_stream
			if pending.RetryCount > 5 {
				val := t.client.XRangeN(ctx, streamKey, pending.ID, "+", 1).Val()
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
				r.Topic = msg.TopicName
				r.Channel = t.channel
				r.MoodType = msg.MoodType

				r.Level = ErrLevel
				r.Info = "too long pending"

				if err := t.log.saveLog(ctx, r); err != nil {
					logger.New().Error(err)
				}

				if err := t.client.XDel(ctx, streamKey, val[0].ID).Err(); err != nil {
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
				r := t.execute(ctx, &nv)

				group := t.errGroupPool.Get().(*errgroup.Group)
				group.TryGo(func() error {
					return t.ack(ctx, stream, channel, nv.ID)
				})
				group.TryGo(func() error {
					return t.log.saveLog(ctx, r)
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
	err := t.client.XAck(ctx, stream, channel, ids...).Err()
	// delete data from `stream`
	err = t.client.XDel(ctx, stream, ids...).Err()
	return err

}

func (t *RedisHandle) execute(ctx context.Context, message *redis.XMessage) *ConsumerResult {

	r := t.result.Get().(*ConsumerResult)

	// var cancel context.CancelFunc
	nctx, cancel := context.WithTimeout(context.Background(), t.timeOut)

	defer func() {
		r = &ConsumerResult{Level: InfoLevel, Info: SuccessInfo, RunTime: ""}
		t.result.Put(r)
		cancel()
	}()

	msg := messageToStruct(message)

	r.Id = msg.Id
	r.BeginTime = time.Now()

	retryCount, err := RetryInfo(nctx, func() error {
		return t.run.(RunSubscribe).Run(nctx, msg)
	}, t.jobMaxRetry)

	r.EndTime = time.Now()
	sub := r.EndTime.Sub(r.BeginTime)
	r.AddTime = msg.AddTime
	r.Retry = retryCount
	r.Payload = msg.Payload
	r.Priority = msg.Priority
	r.RunTime = sub.String()
	r.ExecuteTime = msg.ExecuteTime
	r.Topic = msg.TopicName
	r.Channel = t.channel
	r.MoodType = msg.MoodType

	if err != nil {
		t.run.(RunSubscribe).Error(err)
		r.Level = ErrLevel
		r.Info = FlagInfo(err.Error())
	}
	return r
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
