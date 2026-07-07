package bredis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/spf13/cast"
)

type Normal struct {
	base   Base
	maxLen int64
}

func NewNormal(client redis.UniversalClient, prefix string, maxLen int64, consumerCount int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config) *Normal {

	return &Normal{
		maxLen: maxLen,
		base: Base{
			client:           client,
			processLogger:    NewProcessLog(client, prefix),
			subType:          btype.NormalSubscribe,
			prefix:           prefix,
			deadLetterIdle:   deadLetterIdle,
			blockDuration:    DefaultBlockDuration,
			consumers:        consumerCount,
			consumerPoolSize: consumerPoolSize,
			captureConfig:    config,
		},
	}
}

func (t *Normal) Enqueue(ctx context.Context, data map[string]any) error {
	channel := ""
	topic := ""

	if v, ok := data["channel"]; ok {
		channel = cast.ToString(v)
	}
	if v, ok := data["topic"]; ok {
		topic = cast.ToString(v)
	}

	stream := tool.MakeStreamKey(t.base.subType, t.base.prefix, channel, topic)
	args := NewZAddArgs(stream, "", "*", t.maxLen, 0, data)

	err := t.base.client.XAdd(ctx, args).Err()
	if err != nil {
		return fmt.Errorf("[RedisBroker.enqueue] normal xadd error:%w", err)
	}

	return nil
}

func (t *Normal) Dequeue(ctx context.Context, channel, topic string, do public.CallbackWithRetry) {
	go func() {
		t.base.DeadLetter(ctx, channel, topic)
	}()
	t.base.Dequeue(ctx, channel, topic, do)
}
