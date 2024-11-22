package bredis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
)

type RdbConsumer struct {
	client redis.UniversalClient
}

func (t *RdbConsumer) Read(ctx context.Context, subscribeType btype.SubscribeType, prefix, channel, topic string, minConsumers int64) {

	//streamKey := tool.MakeStreamKey(subscribeType, prefix, channel, topic)
	//readGroupArgs := redisx.NewReadGroupArgs(channel, streamKey, []string{streamKey, ">"}, minConsumers, 10*time.Second)
	//stream, err := t.client.XReadGroup(ctx, readGroupArgs).Result()
	//
	//if err != nil {
	//
	//}
}
