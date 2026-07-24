package bredis

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/spf13/cast"
)

type ProcessLog struct {
	client                  redis.UniversalClient
	prefix                  string
	normalQueuePartitions   int64
	sequenceQueuePartitions int64
}

func NewProcessLog(client redis.UniversalClient, prefix string, sequenceQueuePartitions ...int64) *ProcessLog {
	partitions := int64(0)
	if len(sequenceQueuePartitions) > 0 {
		partitions = sequenceQueuePartitions[0]
	}
	return NewProcessLogWithPartitions(client, prefix, 0, partitions)
}

func NewProcessLogWithPartitions(client redis.UniversalClient, prefix string, normalQueuePartitions, sequenceQueuePartitions int64) *ProcessLog {
	if normalQueuePartitions > 0 {
		normalQueuePartitions = normalizePartitionCount(normalQueuePartitions)
	}
	if sequenceQueuePartitions > 0 {
		sequenceQueuePartitions = normalizeSequenceQueuePartitionCount(sequenceQueuePartitions)
	}
	return &ProcessLog{
		client:                  client,
		prefix:                  prefix,
		normalQueuePartitions:   normalQueuePartitions,
		sequenceQueuePartitions: sequenceQueuePartitions,
	}
}

func (t *ProcessLog) AddLog(ctx context.Context, data map[string]any) error {

	logStream := tool.MakeLogicKey(t.prefix)

	moodType := btype.NORMAL

	if v, ok := data["moodType"]; ok {
		moodType = btype.MoodType(cast.ToString(v))
	}

	if key := t.statusKey(data, moodType); key != "" {
		if err := SaveHSetScript.Run(ctx, t.client, []string{key}, data).Err(); err != nil {
			return err
		}
	}

	data["logType"] = bstatus.Logic
	// write job log into redis
	if err := t.client.XAdd(ctx, &redis.XAddArgs{
		Stream:     logStream,
		NoMkStream: false,
		MaxLen:     20000,
		Approx:     false,
		ID:         "*",
		Values:     data,
	}).Err(); err != nil {
		return err
	}

	return nil
}

func (t *ProcessLog) statusKey(data map[string]any, moodType btype.MoodType) string {
	channel := cast.ToString(data["channel"])
	topic := cast.ToString(data["topic"])
	id := cast.ToString(data["id"])
	if channel == "" || topic == "" || id == "" {
		return ""
	}

	switch moodType {
	case btype.NORMAL:
		if t.normalQueuePartitions <= 0 {
			return ""
		}
		topology := newNormalQueueTopology(t.prefix, channel, topic, t.normalQueuePartitions)
		partition := topology.partition(id)
		return topology.messageStatusKey(partition, id)
	case btype.DELAY:
		if t.normalQueuePartitions <= 0 {
			return ""
		}
		topology := newDelayQueueTopology(t.prefix, channel, topic, t.normalQueuePartitions)
		partition := topology.partition(id)
		return topology.messageStatusKey(partition, id)
	case btype.SEQUENCE_QUEUE:
		return t.sequenceQueueStatusKey(channel, topic, cast.ToString(data["orderKey"]), id)
	default:
		return ""
	}
}

func (t *ProcessLog) sequenceQueueStatusKey(channel, topic, orderKey, id string) string {
	if t.sequenceQueuePartitions <= 0 || channel == "" || topic == "" || orderKey == "" || id == "" {
		return tool.MakeStatusKey(t.prefix, channel, topic, id)
	}
	topology := newSequenceQueueTopology(t.prefix, channel, topic, t.sequenceQueuePartitions)
	partition := topology.partition(orderKey)
	return topology.messageStatusKey(partition, orderKey, id)
}
