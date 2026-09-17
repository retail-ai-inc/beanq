package routers

import (
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
)

type Schedule struct {
	client redis.UniversalClient
	prefix string
}

func NewSchedule(client redis.UniversalClient, prefix string) *Schedule {
	return &Schedule{client: client, prefix: prefix}
}

func (t *Schedule) List(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	ctx := r.Context()

	legacyKeys, _, err := scanKeys(ctx, t.client, strings.Join([]string{t.prefix, "*", "delay_stream:stream"}, ":"), 0)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	v2Keys, _, err := scanKeys(ctx, t.client, strings.Join([]string{t.prefix, "*", "*", "*", "delay_queue", "scheduled*"}, ":"), 0)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	keys := append(legacyKeys, v2Keys...)

	data := make(map[string][]Stream, 0)
	for _, queue := range keys {

		arr := strings.Split(queue, ":")
		if len(arr) < 4 {
			continue
		}
		channel, topic, mood := arr[1], arr[2], arr[3]
		channel = strings.Trim(channel, "{}")
		topic = strings.Trim(topic, "{}")
		if strings.Contains(queue, ":delay_queue:scheduled") {
			// v2: prefix:channel:topic:{partition}:delay_queue:scheduledNNN
			if len(arr) < 6 {
				continue
			}
			mood = "delay"
		}

		var size int64
		if strings.Contains(queue, ":delay_queue:scheduled") {
			size, err = t.client.ZCard(ctx, queue).Result()
		} else {
			size, err = t.client.XLen(ctx, queue).Result()
		}
		if err != nil {
			continue
		}
		stream := Stream{
			Prefix:   arr[0],
			Channel:  channel,
			Topic:    topic,
			MoodType: mood,
			State:    "Run",
			Size:     int(size),
		}
		data[channel] = append(data[channel], stream)
	}

	result.Data = data
	_ = result.Json(w, http.StatusOK)
}
