package routers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/spf13/cast"
)

type ObjectStruct struct {
	ValueAt          string
	Encoding         string
	RefCount         int
	SerizlizedLength int
	Lru              int
	LruSecondsIdle   int
}

func Object(ctx context.Context, client redis.UniversalClient, queueName string) (objstr ObjectStruct) {

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

func HGetAll(ctx context.Context, client redis.UniversalClient, key string) (map[string]string, error) {
	return client.HGetAll(ctx, key).Result()
}

func HSet(ctx context.Context, client redis.UniversalClient, key string, data map[string]any) error {
	return client.HSet(ctx, key, data).Err()
}

func Del(ctx context.Context, client redis.UniversalClient, key string) error {
	return client.Del(ctx, key).Err()
}

func scanKeys(ctx context.Context, client redis.UniversalClient, pattern string, limit int64) ([]string, uint64, error) {
	keys := make([]string, 0)
	var cursor uint64
	for {
		batchSize := int64(250)
		if limit > 0 && limit-int64(len(keys)) < batchSize {
			batchSize = limit - int64(len(keys))
		}
		if batchSize <= 0 {
			return keys, cursor, nil
		}
		batch, next, err := client.Scan(ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			return nil, cursor, err
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 || (limit > 0 && int64(len(keys)) >= limit) {
			return keys, cursor, nil
		}
	}
}

func ZScan(ctx context.Context, client redis.UniversalClient, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return client.ZScan(ctx, key, cursor, match, count).Result()
}

func XRevRange(ctx context.Context, client redis.UniversalClient, stream, start, stop string) ([]redis.XMessage, error) {
	return client.XRevRange(ctx, stream, start, stop).Result()
}

func ZRemRangeByScore(ctx context.Context, client redis.UniversalClient, key, min, max string) error {
	return client.ZRemRangeByScore(ctx, key, min, max).Err()
}

func XRangeN(ctx context.Context, client redis.UniversalClient, stream string, start, stop string, count int64) ([]redis.XMessage, error) {
	return client.XRangeN(ctx, stream, start, stop, count).Result()
}

func ZRange(ctx context.Context, client redis.UniversalClient, match string, page, pageSize int64) (map[string]any, error) {

	cmd := client.ZRange(ctx, match, page, pageSize)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}

	result, err := cmd.Result()
	if err != nil {
		return nil, err
	}

	njson := json.Json

	length, err := client.ZLexCount(ctx, match, "-", "+").Result()
	if err != nil {
		return nil, err
	}
	d := make([]map[string]any, 0, pageSize)
	for _, v := range result {

		cmd := client.ZRank(ctx, match, v)
		key, err := cmd.Result()
		if err != nil {
			continue
		}
		payloadByte := []byte(v)
		npayload := njson.Get(payloadByte, "Payload")
		addTime := njson.Get(payloadByte, "AddTime")
		runTime := njson.Get(payloadByte, "RunTime")
		group := njson.Get(payloadByte, "Group")

		queuestr := njson.Get(payloadByte, "Queue").ToString()
		queues := strings.Split(queuestr, ":")
		queue := queuestr
		if len(queues) >= 4 {
			queue = queues[2]
		}

		ttl := time.Until(cast.ToTime(njson.Get(payloadByte, "ExpireTime").ToString())).Seconds()
		d = append(d, map[string]any{"key": key, "ttl": fmt.Sprintf("%.3f", ttl), "addTime": addTime, "runTime": runTime, "group": group, "queue": queue, "payload": npayload})

	}
	return map[string]any{"data": d, "total": length}, nil
}

type Msg struct {
	Id      string `json:"id"`
	Level   string
	Info    string
	Payload any `json:"payload"`

	AddTime     string    `json:"addTime"`
	ExpireTime  time.Time `json:"expireTime"`
	RunTime     string    `json:"runTime"`
	BeginTime   time.Time
	EndTime     time.Time
	ExecuteTime time.Time
	Topic       string `json:"topic"`
	Channel     string `json:"channel"`
	Consumer    string `json:"consumer"`
	Score       string
}

type Stream struct {
	Prefix     string `json:"prefix"`
	Channel    string `json:"channel"`
	Topic      string `json:"topic"`
	MoodType   string `json:"moodType"`
	State      string `json:"state"`
	Size       int    `json:"size"`
	Idle       int    `json:"idle"`
	Partitions int    `json:"partitions,omitempty"`
	Legacy     bool   `json:"legacy,omitempty"`
}

func QueueInfo(ctx context.Context, client redis.UniversalClient, prefix string) (any, error) {

	// get queues
	queues, _, err := scanKeys(ctx, client, QueueKey(prefix), 0)
	if err != nil {
		return nil, err
	}

	data := make(map[string][]Stream, 0)
	partitioned := make(map[string]Stream)
	for _, queue := range queues {
		if stream, key, ok, streamErr := partitionedQueueStream(ctx, client, queue, prefix); streamErr != nil {
			return nil, streamErr
		} else if ok {
			current := partitioned[key]
			if current.Partitions == 0 {
				current = stream
				current.Size = 0
			}
			current.Size += stream.Size
			current.Partitions++
			if stream.Idle < current.Idle || current.Idle == 0 {
				current.Idle = stream.Idle
			}
			partitioned[key] = current
			continue
		}

		arr := strings.Split(queue, ":")
		if len(arr) < 4 {
			continue
		}
		keyType, err := client.Type(ctx, queue).Result()
		if err != nil {
			return nil, err
		}
		if keyType != "stream" {
			continue
		}
		arr[1] = strings.ReplaceAll(arr[1], "{", "")
		arr[2] = strings.ReplaceAll(arr[2], "}", "")
		size, err := client.XLen(ctx, queue).Result()
		if err != nil {
			return nil, err
		}

		stream := Stream{
			Prefix:   arr[0],
			Channel:  arr[1],
			Topic:    arr[2],
			MoodType: legacyQueueMoodType(arr[3]),
			State:    "Run",
			Size:     int(size),
			Legacy:   arr[3] == "normal_stream",
		}
		data[arr[1]] = append(data[arr[1]], stream)
	}
	for _, stream := range partitioned {
		data[stream.Channel] = append(data[stream.Channel], stream)
	}

	return data, nil
}
func ScheduleQueueKey(prefix string) string {
	return strings.Join([]string{prefix, "*", "zset"}, ":")
}
func QueueKey(prefix string) string {
	return "*" + prefix + "*:stream*"
}

func partitionedQueueStream(ctx context.Context, client redis.UniversalClient, key, prefix string) (Stream, string, bool, error) {
	stream, ok := partitionedQueueRoute(key, prefix)
	if !ok {
		return Stream{}, "", false, nil
	}
	size, err := client.XLen(ctx, key).Result()
	if err != nil {
		return Stream{}, "", false, err
	}
	stream.Size = int(size)
	return stream, stream.Channel + "\x00" + stream.Topic + "\x00" + stream.MoodType, true, nil
}

func partitionedQueueRoute(key, prefix string) (Stream, bool) {
	parts := strings.Split(key, ":")
	marker := -1
	moodType := ""
	for i, part := range parts {
		if i < 3 || i+1 >= len(parts) {
			continue
		}
		switch part {
		case "normal_queue":
			moodType = string(btype.NORMAL)
		case "delay_queue":
			moodType = string(btype.DELAY)
		case "sequence_queue":
			moodType = string(btype.SEQUENCE_QUEUE)
		default:
			continue
		}
		isStream := (part == "normal_queue" || part == "delay_queue") && i+2 == len(parts)
		isScheduler := part == "sequence_queue" && i+3 == len(parts) && parts[i+2] == "scheduler"
		if strings.HasPrefix(parts[i+1], "stream") && (isStream || isScheduler) {
			marker = i
			break
		}
	}
	if marker < 0 || parts[0] != prefix {
		return Stream{}, false
	}
	return Stream{Prefix: prefix, Channel: parts[1], Topic: parts[2], MoodType: moodType, State: "Run"}, true
}

func legacyQueueMoodType(streamType string) string {
	switch streamType {
	case "normal_stream":
		return string(btype.NORMAL)
	case "delay_stream":
		return string(btype.DELAY)
	default:
		return streamType
	}
}

func ReturnHtml(w http.ResponseWriter, errorString string) {
	html := `<html>
	<head>
		<meta charset="UTF-8">
		<title>Error</title>
	</head>
	<body>
		<div style="text-align:center;font-weight:bold;margin-top:40vh;">%s</div>
	</body></html>`

	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	nhtml := fmt.Sprintf(html, errorString)
	//#nosec G705
	_, _ = w.Write([]byte(nhtml))
	w.WriteHeader(http.StatusInternalServerError)
}
