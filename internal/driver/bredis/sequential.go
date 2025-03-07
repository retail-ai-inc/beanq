package bredis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	public "github.com/retail-ai-inc/beanq/v3/internal"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

type Sequential struct {
	base Base
}

func NewSequential(client redis.UniversalClient, prefix string, consumerCount int64, deadLetterIdle time.Duration) *Sequential {

	return &Sequential{
		base: Base{
			client:         client,
			IProcessLog:    NewProcessLog(client, prefix),
			subType:        btype.SequentialSubscribe,
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

func (t *Sequential) Enqueue(ctx context.Context, data map[string]any) error {
	
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

func (t *Sequential) Dequeue(ctx context.Context, channel, topic string, do public.CallBack) {

	go func() {
		t.base.DeadLetter(ctx, channel, topic)
	}()
	t.base.Dequeue(ctx, channel, topic, do)

}
