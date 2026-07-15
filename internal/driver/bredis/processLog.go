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
	sequenceQueuePartitions int64
}

func NewProcessLog(client redis.UniversalClient, prefix string, sequenceQueuePartitions ...int64) *ProcessLog {
	partitions := int64(0)
	if len(sequenceQueuePartitions) > 0 {
		partitions = normalizeSequenceQueuePartitionCount(sequenceQueuePartitions[0])
	}
	return &ProcessLog{
		client:                  client,
		prefix:                  prefix,
		sequenceQueuePartitions: partitions,
	}
}

func (t *ProcessLog) AddLog(ctx context.Context, data map[string]any) error {

	logStream := tool.MakeLogicKey(t.prefix)

	moodType := btype.NORMAL

	if v, ok := data["moodType"]; ok {
		moodType = btype.MoodType(cast.ToString(v))
	}

	if moodType == btype.SEQUENCE_QUEUE {

		channel, id, topic, orderKey := "", "", "", ""
		if v, ok := data["channel"]; ok {
			channel = cast.ToString(v)
		}
		if v, ok := data["id"]; ok {
			id = cast.ToString(v)
		}
		if v, ok := data["topic"]; ok {
			topic = cast.ToString(v)
		}
		if v, ok := data["orderKey"]; ok {
			orderKey = cast.ToString(v)
		}

		key := t.sequenceQueueStatusKey(channel, topic, orderKey, id)
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

func (t *ProcessLog) sequenceQueueStatusKey(channel, topic, orderKey, id string) string {
	if t.sequenceQueuePartitions <= 0 || channel == "" || topic == "" || orderKey == "" || id == "" {
		return tool.MakeStatusKey(t.prefix, channel, topic, id)
	}
	topology := newSequenceQueueTopology(t.prefix, channel, topic, t.sequenceQueuePartitions)
	partition := topology.partition(orderKey)
	return topology.messageStatusKey(partition, orderKey, id)
}
