package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"net/http"
	"strings"
)

type Schedule struct {
	client redis.UniversalClient
	prefix string
}

func NewSchedule(client redis.UniversalClient, prefix string) *Schedule {
	return &Schedule{client: client, prefix: prefix}
}

func (t *Schedule) List(bctx *bwebframework.BeanContext) error {

	result, cancel := response.Get()
	defer cancel()

	ctx := bctx.Request.Context()

	nodeId := bctx.Request.Header.Get("X-Cluster-Nodeid")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	key := strings.Join([]string{t.prefix, "*", "delay_stream:stream"}, ":")

	keys, err := client.Keys(ctx, key)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(bctx.Writer, http.StatusInternalServerError)
	}

	data := make(map[string][]Stream, 0)
	for _, queue := range keys {

		arr := strings.Split(queue, ":")
		if len(arr) < 4 {
			continue
		}
		arr[1] = strings.ReplaceAll(arr[1], "{", "")
		arr[2] = strings.ReplaceAll(arr[2], "}", "")

		obj, err := client.Object(ctx, queue)
		if err != nil {
			continue
		}
		stream := Stream{
			Prefix:   arr[0],
			Channel:  arr[1],
			Topic:    arr[2],
			MoodType: arr[3],
			State:    "Run",
			Size:     obj.SerizlizedLength,
			Idle:     obj.LruSecondsIdle,
		}
		data[arr[1]] = append(data[arr[1]], stream)
	}

	result.Data = data
	return result.Json(bctx.Writer, http.StatusOK)
}
