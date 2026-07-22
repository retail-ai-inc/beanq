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

	key := strings.Join([]string{t.prefix, "*", "delay_stream:stream"}, ":")

	keys, _, err := scanKeys(ctx, t.client, key, 0)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}

	data := make(map[string][]Stream, 0)
	for _, queue := range keys {

		arr := strings.Split(queue, ":")
		if len(arr) < 4 {
			continue
		}
		arr[1] = strings.ReplaceAll(arr[1], "{", "")
		arr[2] = strings.ReplaceAll(arr[2], "}", "")

		size, err := t.client.XLen(ctx, queue).Result()
		if err != nil {
			continue
		}
		stream := Stream{
			Prefix:   arr[0],
			Channel:  arr[1],
			Topic:    arr[2],
			MoodType: arr[3],
			State:    "Run",
			Size:     int(size),
		}
		data[arr[1]] = append(data[arr[1]], stream)
	}

	result.Data = data
	_ = result.Json(w, http.StatusOK)
}
