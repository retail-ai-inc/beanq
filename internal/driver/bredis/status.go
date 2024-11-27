package bredis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
)

type Status struct {
	client redis.UniversalClient
	prefix string
}

func NewStatus(client redis.UniversalClient, prefix string) *Status {
	return &Status{
		client: client,
		prefix: prefix,
	}
}

func (t *Status) Status(ctx context.Context, channel, topic, id string) (map[string]string, error) {

	key := tool.MakeStatusKey(t.prefix, channel, topic, id)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:

			cmd := t.client.HGetAll(ctx, key)

			if err := cmd.Err(); err != nil {
				return nil, err
			}

			val := cmd.Val()
			if len(val) <= 0 {
				continue
			}

			if v, ok := val["status"]; ok {
				if v != bstatus.StatusSuccess && v != bstatus.StatusFailed {
					continue
				}
			}

			return val, nil
		}
	}
}
