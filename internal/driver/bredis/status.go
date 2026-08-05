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
	normalQueuePartitions   int64
	sequenceQueuePartitions int64
}

func NewStatus(client redis.UniversalClient, prefix string, sequenceQueuePartitions ...int64) *Status {
	partitions := int64(0)
	if len(sequenceQueuePartitions) > 0 {
		partitions = sequenceQueuePartitions[0]
	}
	return NewStatusWithPartitions(client, prefix, 0, partitions)
}

func NewStatusWithPartitions(client redis.UniversalClient, prefix string, normalQueuePartitions, sequenceQueuePartitions int64) *Status {
	if normalQueuePartitions > 0 {
		normalQueuePartitions = normalizePartitionCount(normalQueuePartitions)
	}
	if sequenceQueuePartitions > 0 {
		sequenceQueuePartitions = normalizeSequenceQueuePartitionCount(sequenceQueuePartitions)
	}
	return &Status{
		client:                  client,
		prefix:                  prefix,
		normalQueuePartitions:   normalQueuePartitions,
		sequenceQueuePartitions: sequenceQueuePartitions,
	}
}

func (t *Status) Status(ctx context.Context, channel, topic, id string) (map[string]string, error) {
	if t.normalQueuePartitions <= 0 || id == "" {
		return t.waitStatus(ctx, tool.MakeStatusKey(t.prefix, channel, topic, id))
	}
	normalTopology := newNormalQueueTopology(t.prefix, channel, topic, t.normalQueuePartitions)
	delayTopology := newDelayQueueTopology(t.prefix, channel, topic, t.normalQueuePartitions)
	partition := normalTopology.partition(id)
	return t.waitStatuses(ctx,
		normalTopology.messageStatusKey(partition, id),
		delayTopology.messageStatusKey(partition, id),
	)
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
	return t.waitStatuses(ctx, key)
}

func (t *Status) waitStatuses(ctx context.Context, keys ...string) (map[string]string, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			for _, key := range keys {
				cmd := t.client.HGetAll(ctx, key)
				if err := cmd.Err(); err != nil {
					return nil, err
				}
				val := cmd.Val()
				if len(val) <= 0 {
					continue
				}
				if v, ok := val["status"]; ok && v != bstatus.StatusSuccess && v != bstatus.StatusFailed {
					continue
				}
				return val, nil
			}
		}
	}
}
