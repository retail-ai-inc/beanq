package bredis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	jsonhelper "github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/spf13/cast"
)

const delayQueuePromoteBatch = int64(100)

type delayQueueStore struct {
	client   redis.UniversalClient
	topology delayQueueTopology
	maxLen   int64
	wait     replicationWait
}

func newDelayQueueStore(client redis.UniversalClient, topology delayQueueTopology, maxLen int64, waits ...replicationWait) *delayQueueStore {
	if maxLen <= 0 {
		maxLen = partitionQueueDefaultMaxLen
	}
	return &delayQueueStore{client: client, topology: topology, maxLen: maxLen, wait: firstReplicationWait(waits)}
}

func (s *delayQueueStore) metadata() partitionQueueMetadata {
	return partitionQueueMetadata{schema: delayQueueSchemaVersion, partitions: s.topology.partitions, capacity: s.maxLen}
}

func (s *delayQueueStore) ensureMetadata(ctx context.Context) error {
	return ensurePartitionQueueMetadata(ctx, s.client, s.wait, s.topology.metadataKey(), "delay queue", s.metadata())
}

func (s *delayQueueStore) bootstrapGroups(ctx context.Context, group string) error {
	return bootstrapPartitionGroups(ctx, s.client, group, "delay queue", s.topology.partitions, s.topology.streamKey)
}

func (s *delayQueueStore) bootstrapGroup(ctx context.Context, group string, partition int64) error {
	return bootstrapPartitionGroup(ctx, s.client, group, "delay queue", partition, s.topology.streamKey(partition))
}

func (s *delayQueueStore) enqueue(ctx context.Context, data map[string]any, wait replicationWait) error {
	messageID := cast.ToString(data["id"])
	if messageID == "" {
		return errors.New("delay queue message id is required")
	}
	payload, err := jsonhelper.Marshal(data)
	if err != nil {
		return err
	}
	partition := s.topology.partition(messageID)
	executeTime := cast.ToTime(data["executeTime"])
	priority := cast.ToFloat64(data["priority"])
	// Earlier execution times sort first; priority only orders messages within the same millisecond.
	score := float64(executeTime.UnixMilli()) - priority/1e3
	if !wait.enabled() {
		if err := s.client.ZAdd(ctx, s.topology.scheduledKey(partition), redis.Z{Score: score, Member: payload}).Err(); err != nil {
			return fmt.Errorf("delay partition %d zadd: %w", partition, err)
		}
		return nil
	}
	scheduled := s.topology.scheduledKey(partition)
	_, err = wait.execute(ctx, s.client, scheduled, func(pipe redis.Pipeliner) redis.Cmder {
		return pipe.ZAdd(ctx, scheduled, redis.Z{Score: score, Member: payload})
	})
	if err != nil {
		return fmt.Errorf("delay partition %d zadd: %w", partition, err)
	}
	return nil
}

func (s *delayQueueStore) promote(ctx context.Context, partition int64, now time.Time) (bool, error) {
	zset := s.topology.scheduledKey(partition)
	stream := s.topology.streamKey(partition)
	max := fmt.Sprintf("%d", now.UnixMilli()+1)
	var promoted bool
	err := s.client.Watch(ctx, func(tx *redis.Tx) error {
		values, err := tx.ZRangeArgs(ctx, redis.ZRangeArgs{
			Key:     zset,
			Start:   "0",
			Stop:    max,
			ByScore: true,
			Offset:  0,
			Count:   delayQueuePromoteBatch,
		}).Result()
		if err != nil || len(values) == 0 {
			return err
		}
		decoded := make([]map[string]any, 0, len(values))
		for _, value := range values {
			data := make(map[string]any)
			if err := jsonhelper.Unmarshal([]byte(value), &data); err != nil {
				continue
			}
			decoded = append(decoded, data)
		}
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			for _, data := range decoded {
				pipe.XAdd(ctx, NewZAddArgs(stream, "", "*", s.maxLen, 0, data))
			}
			// Invalid entries are removed too; otherwise they permanently block the head.
			members := make([]any, len(values))
			for i := range values {
				members[i] = values[i]
			}
			pipe.ZRem(ctx, zset, members...)
			return nil
		})
		if err != nil {
			return err
		}
		if s.wait.enabled() {
			if err := s.wait.confirm(ctx, tx); err != nil {
				return err
			}
		}
		promoted = len(decoded) > 0
		return nil
	}, zset, stream)
	if errors.Is(err, redis.TxFailedErr) {
		return false, nil
	}
	return promoted, err
}
