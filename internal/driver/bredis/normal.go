package bredis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/retail-ai-inc/beanq/v3/internal"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

type Normal struct {
	base   Base
	maxLen int64
}

func NewNormal(client redis.UniversalClient, prefix string, maxLen int64, consumerCount int64, deadLetterIdle time.Duration) *Normal {

	return &Normal{
		maxLen: maxLen,
		base: Base{
			client:         client,
			IProcessLog:    NewProcessLog(client, prefix),
			subType:        btype.NormalSubscribe,
			prefix:         prefix,
			deadLetterIdle: deadLetterIdle,
			blockDuration:  DefaultBlockDuration,
			errGroup: sync.Pool{New: func() any {
				return new(errgroup.Group)
			}},
			consumers: consumerCount,
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

func (t *Normal) Dequeue(ctx context.Context, channel, topic string, do public.CallBack) {

	go func() {
		t.base.DeadLetter(ctx, channel, topic)
	}()
	t.base.Dequeue(ctx, channel, topic, do)

}
