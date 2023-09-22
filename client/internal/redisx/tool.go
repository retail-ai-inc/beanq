package redisx

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

type ObjectStruct struct {
	ValueAt          string
	RefCount         int
	Encoding         string
	SerizlizedLength int
	Lru              int
	LruSecondsIdle   int
}

func Object(ctx context.Context, client *redis.Client, queueName string) (objstr ObjectStruct) {
	obj := client.DebugObject(ctx, queueName)

	str, _ := obj.Result()
	// Value at:0x7fc38fe77cc0 refcount:1 encoding:stream serializedlength:12 lru:7878503 lru_seconds_idle:3
	valueAt := "Value at"
	if strings.HasPrefix(str, valueAt) {
		str = strings.ReplaceAll(str, valueAt, "value_at")
	}

	strs := strings.Split(str, " ")

	for _, s := range strs {
		sarr := strings.Split(s, ":")
		if len(sarr) >= 2 {
			switch sarr[0] {
			case "value_at":
				objstr.ValueAt = sarr[1]
			case "refcount":
				objstr.RefCount = cast.ToInt(sarr[1])
			case "encoding":
				objstr.Encoding = sarr[1]
			case "serializedlength":
				objstr.SerizlizedLength = cast.ToInt(sarr[1])
			case "lru":
				objstr.Lru = cast.ToInt(sarr[1])
			case "lru_seconds_idle":
				objstr.LruSecondsIdle = cast.ToInt(sarr[1])
			}
		}
	}
	return
}
func Keys(ctx context.Context, client *redis.Client, key string) ([]string, error) {
	cmd := client.Keys(ctx, key)
	queues, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	return queues, nil
}
func Info(ctx context.Context, client *redis.Client) (map[string]string, error) {
	infoStr, err := client.Info(ctx).Result()
	if err != nil {
		return nil, err
	}
	info := make(map[string]string)
	lines := strings.Split(infoStr, "\r\n")
	for _, l := range lines {
		kv := strings.Split(l, ":")
		if len(kv) == 2 {
			info[kv[0]] = kv[1]
		}
	}
	return info, nil

}
func ScheduleQueueKey(prefix string) string {
	return strings.Join([]string{prefix, "*", "zset"}, ":")
}
func QueueKey(prefix string) string {
	return strings.Join([]string{prefix, "*", "stream"}, ":")
}
