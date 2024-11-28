package bredis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/spf13/cast"
)

type ProcessLog struct {
	client redis.UniversalClient
	prefix string
}

func NewProcessLog(client redis.UniversalClient, prefix string) *ProcessLog {
	return &ProcessLog{
		client: client,
		prefix: prefix,
	}
}

func (t *ProcessLog) AddLog(ctx context.Context, data map[string]any) error {

	logStream := tool.MakeLogicKey(t.prefix)

	moodType := btype.NORMAL

	if v, ok := data["moodType"]; ok {
		moodType = btype.MoodType(cast.ToString(v))
	}

	if moodType == btype.SEQUENTIAL {

		channel, id, topic := "", "", ""
		if v, ok := data["channel"]; ok {
			channel = cast.ToString(v)
		}
		if v, ok := data["id"]; ok {
			id = cast.ToString(v)
		}
		if v, ok := data["topic"]; ok {
			topic = cast.ToString(v)
		}

		key := tool.MakeStatusKey(t.prefix, channel, topic, id)
		if err := SaveHSetScript.Run(ctx, t.client, []string{key}, data).Err(); err != nil {
			return err
		}
	}

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
