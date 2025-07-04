package bredis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/spf13/cast"
)

type Sequence struct {
	base Base
}

func NewSequence(client redis.UniversalClient, prefix string, consumerCount int64, consumerPoolSize int, deadLetterIdle time.Duration, config *capture.Config) *Sequence {

	return &Sequence{
		base: Base{
			client:           client,
			IProcessLog:      NewProcessLog(client, prefix),
			subType:          btype.SequentialSubscribe,
			prefix:           prefix,
			deadLetterIdle:   deadLetterIdle,
			blockDuration:    DefaultBlockDuration,
			consumers:        consumerCount,
			consumerPoolSize: consumerPoolSize,
			captureConfig:    config,
		},
	}
}
func (t *Sequence) ForceUnlock(_ context.Context, channel, topic, orderKey string) error {

	return nil

}

func (t *Sequence) Enqueue(ctx context.Context, data map[string]any) error {

	channel := ""
	topic := ""
	id := ""

	if v, ok := data["channel"]; ok {
		channel = cast.ToString(v)
	}
	if v, ok := data["topic"]; ok {
		topic = cast.ToString(v)
	}
	if v, ok := data["id"]; ok {
		id = cast.ToString(v)
	}

	streamKey := tool.MakeStreamKey(t.base.subType, t.base.prefix, channel, topic)

	key := tool.MakeStatusKey(t.base.prefix, channel, topic, id)

	exist, err := HashDuplicateIdScript.Run(ctx, t.base.client, []string{key, streamKey}, data).Bool()

	if err != nil {
		return err
	}
	if exist {
		return fmt.Errorf("idempotency check: %w", bstatus.ErrIdempotent)
	}

	return nil
}

func (t *Sequence) Dequeue(ctx context.Context, channel, topic string, do public.CallBack) {

	go func() {
		t.base.DeadLetter(ctx, channel, topic)
	}()
	t.base.Dequeue(ctx, channel, topic, do)

}
