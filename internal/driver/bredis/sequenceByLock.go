package bredis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/spf13/cast"
)

type SequenceByLock struct {
	base Base
}

func NewSequenceByLock(client redis.UniversalClient, prefix string, consumerCount int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config) *SequenceByLock {
	base := Base{
		client:           client,
		IProcessLog:      NewProcessLog(client, prefix),
		subType:          btype.SequentialByLockSubscribe,
		prefix:           prefix,
		deadLetterIdle:   deadLetterIdle,
		blockDuration:    DefaultBlockDuration,
		consumers:        consumerCount,
		consumerPoolSize: consumerPoolSize,
		captureConfig:    config,
	}
	return &SequenceByLock{base: base}
}

func (t *SequenceByLock) ForceUnlock(ctx context.Context, channel, topic, orderKey string) error {

	key := tool.MakeSequenceLockKey(t.base.prefix, channel, topic, orderKey)
	return t.base.client.Del(ctx, key).Err()

}

func (t *SequenceByLock) Enqueue(ctx context.Context, data map[string]any) error {

	channel, topic, orderKey, lockOrderKeyTTL := "", "", "", time.Duration(0)

	if v, ok := data["channel"]; ok {
		channel = cast.ToString(v)
	}
	if v, ok := data["topic"]; ok {
		topic = cast.ToString(v)
	}
	if v, ok := data["orderKey"]; ok {
		orderKey = cast.ToString(v)
	}
	if v, ok := data["lockOrderKeyTTL"]; ok {
		lockOrderKeyTTL = cast.ToDuration(v)
	}
	streamKey := tool.MakeStreamKey(t.base.subType, t.base.prefix, channel, topic)
	orderRediKey := tool.MakeSequenceLockKey(t.base.prefix, channel, topic, orderKey)

	err := SequenceByLockScript.Run(ctx, t.base.client, []string{streamKey, orderRediKey, cast.ToString(lockOrderKeyTTL.Seconds())}, data).Err()
	if err != nil {
		return bstatus.SequentialLockError
	}

	return nil
}

func (t *SequenceByLock) Dequeue(ctx context.Context, channel, topic string, do public.CallbackWithRetry) {

	go func() {
		t.base.DeadLetter(ctx, channel, topic)
	}()
	t.base.Dequeue(ctx, channel, topic, do)

}
