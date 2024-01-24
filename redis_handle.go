package beanq

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"github.com/retail-ai-inc/beanq/helper/stringx"
)

type RedisHandle struct {
	client           redis.UniversalClient
	log              logJobI
	consumer         DoConsumer
	deadLetterTicker *time.Ticker
	channel          string
	topic            string
}

var (
	result = sync.Pool{New: func() any {
		return &ConsumerResult{
			Level:   InfoLevel,
			Info:    SuccessInfo,
			RunTime: "",
		}
	}}

	streamArrayPool = sync.Pool{New: func() any {
		return make([]redis.XStream, 100)
	}}
)

func NewRedisHandle(client redis.UniversalClient, channel, topic string, consumer DoConsumer) *RedisHandle {
	return &RedisHandle{
		client:           client,
		channel:          channel,
		topic:            topic,
		consumer:         consumer,
		log:              newLogJob(client),
		deadLetterTicker: time.NewTicker(100 * time.Second),
	}
}

func (t *RedisHandle) Check(ctx context.Context) error {

	if err := t.checkStream(ctx); err != nil {
		return err
	}
	if err := t.checkDeadletterStream(ctx); err != nil {
		return err
	}
	return nil

}

func (t *RedisHandle) Work(ctx context.Context, done <-chan struct{}) {

	channel := t.channel
	topic := t.topic
	count := Config.MinWorkers
	stream := MakeStreamKey(Config.Redis.Prefix, channel, topic)
	readGroupArgs := redisx.NewReadGroupArgs(channel, stream, []string{stream, ">"}, count, 10*time.Second)

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

// Please refer to http://www.redis.cn/commands/xclaim.html
func (t *RedisHandle) DeadLetter(ctx context.Context, claimDone <-chan struct{}) error {

	streamKey := MakeDeadLetterStreamKey(Config.Redis.Prefix, t.channel, t.topic)
	xAutoClaim := redisx.NewAutoClaimArgs(streamKey, t.channel, Config.DeadLetterIdle, "0-0", 100, t.topic)

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

		streams := streamArrayPool.Get().([]redis.XStream)

		claims, _ := t.client.XAutoClaim(ctx, xAutoClaim).Val()

		if len(claims) > 0 {
			streams = append(streams, redis.XStream{Stream: streamKey, Messages: claims})
			t.do(ctx, streams)
		}
		streams = make([]redis.XStream, 100)
		streamArrayPool.Put(streams)

	}

}

func (t *RedisHandle) do(ctx context.Context, streams []redis.XStream) {

	channel := t.channel
	for key, v := range streams {

		stream := v.Stream
		message := v.Messages

		for _, vv := range message {
			msg, err := parseMapToMessage(vv, stream)
			if err != nil {
				logger.New().Error(err)
				continue
			}
			r, err := t.makeLog(ctx, stream, vv.ID, msg)
			if err != nil {
				logger.New().Error(err)
			}
			r = &ConsumerResult{Level: InfoLevel, Info: SuccessInfo, RunTime: ""}
			result.Put(r)

			if err := t.ack(ctx, stream, channel, vv.ID); err != nil {
				logger.New().Error(err)
			}
		}
		streams[key] = redis.XStream{}
	}
}

func (t *RedisHandle) ack(ctx context.Context, stream, channel string, ids ...string) error {

	// `stream` confirmation message
	err := t.client.XAck(ctx, stream, channel, ids...).Err()
	// delete data from `stream`
	err = t.client.XDel(ctx, stream, ids...).Err()
	return err

}

func (t *RedisHandle) makeLog(ctx context.Context, stream, id string, msg *Message) (*ConsumerResult, error) {

	r := result.Get().(*ConsumerResult)
	r.Id = id
	r.BeginTime = time.Now()
	// if error,then retry to consume
	nerr := make(chan error, 1)
	if err := RetryInfo(func() error {
		defer func() {
			if ne := recover(); ne != nil {
				nerr <- fmt.Errorf("error:%+v,stack:%s", ne, stringx.ByteToString(debug.Stack()))
			}
		}()
		return t.consumer(msg)

	}, Config.JobMaxRetries); err != nil {
		nerr <- err
	}
	select {
	case v := <-nerr:
		if v != nil {
			r.Level = ErrLevel
			r.Info = FlagInfo(v.Error())
		}
	default:

	}
	r.EndTime = time.Now()

	sub := r.EndTime.Sub(r.BeginTime)

	r.Payload = msg.Payload()
	r.RunTime = sub.String()
	r.ExecuteTime = msg.ExecuteTime()
	r.Topic = stream
	r.Channel = t.channel
	// Successfully consumed data, stored in `string`
	if err := t.log.saveLog(ctx, r); err != nil {
		return nil, err
	}

	return r, nil
}

// checkStream   if stream not exist,then create it
func (t *RedisHandle) checkStream(ctx context.Context) error {

	normalStreamKey := MakeStreamKey(Config.Redis.Prefix, t.channel, t.topic)
	normalStreamResult := t.client.XInfoGroups(ctx, normalStreamKey).Val()

	if len(normalStreamResult) < 1 {
		if err := t.client.XGroupCreateMkStream(ctx, normalStreamKey, t.channel, "0").Err(); err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return err
		}
	}
	return nil

}

func (t *RedisHandle) checkDeadletterStream(ctx context.Context) error {

	// if dead letter stream don't exist,then create it
	deadLetterStreamKey := MakeDeadLetterStreamKey(Config.Redis.Prefix, t.channel, t.topic)
	deadLetterStreamResult := t.client.XInfoGroups(ctx, deadLetterStreamKey).Val()

	if len(deadLetterStreamResult) < 1 {
		if err := t.client.XGroupCreateMkStream(ctx, deadLetterStreamKey, t.channel, "0").Err(); err != nil {
			return err
		}
	}
	return nil

}
