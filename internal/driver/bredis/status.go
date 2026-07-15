package bredis

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
)

type Status struct {
	client                  redis.UniversalClient
	prefix                  string
	sequenceQueuePartitions int64
}

func NewStatus(client redis.UniversalClient, prefix string, sequenceQueuePartitions ...int64) *Status {
	partitions := int64(0)
	if len(sequenceQueuePartitions) > 0 {
		partitions = normalizeSequenceQueuePartitionCount(sequenceQueuePartitions[0])
	}
	return &Status{
		client:                  client,
		prefix:                  prefix,
		sequenceQueuePartitions: partitions,
	}
}

func (t *Status) Status(ctx context.Context, channel, topic, id string) (map[string]string, error) {
	key := tool.MakeStatusKey(t.prefix, channel, topic, id)
	return t.waitStatus(ctx, key)
}

func (t *Status) SequenceStatus(ctx context.Context, channel, topic, orderKey, id string) (map[string]string, error) {
	if t.sequenceQueuePartitions <= 0 || orderKey == "" {
		return t.Status(ctx, channel, topic, id)
	}
	topology := newSequenceQueueTopology(t.prefix, channel, topic, t.sequenceQueuePartitions)
	partition := topology.partition(orderKey)
	return t.waitStatus(ctx, topology.messageStatusKey(partition, orderKey, id))
}

func (t *Status) waitStatus(ctx context.Context, key string) (map[string]string, error) {
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
