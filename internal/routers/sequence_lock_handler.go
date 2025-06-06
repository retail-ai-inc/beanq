package routers

import (
	"net/http"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
)

type SequenceLock struct {
	client redis.UniversalClient
	prefix string
}

func NewSequenceLock(client redis.UniversalClient, prefix string) *SequenceLock {
	return &SequenceLock{
		client: client,
		prefix: prefix,
	}
}

func (t *SequenceLock) List(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	ctx := r.Context()
	orderKey := r.URL.Query().Get("orderKey")
	channelName := r.URL.Query().Get("channelName")
	topicName := r.URL.Query().Get("topicName")

	data, err := t.client.HGetAll(ctx, tool.MakeSequenceLockKey(t.prefix, channelName, topicName, orderKey)).Result()
	if err != nil {
		result.Code = response.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	result.Data = data
	_ = result.Json(w, http.StatusOK)
}
func (t *SequenceLock) UnLock(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	keys := strings.Split(r.PathValue("key"), ":")
	if len(keys) < 3 {
		result.Code = response.InternalServerErrorCode
		result.Msg = "key error"
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	
	if err := t.client.Del(r.Context(), tool.MakeSequenceLockKey(t.prefix, keys[0], keys[1], keys[2])).Err(); err != nil {
		result.Code = response.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	_ = result.Json(w, http.StatusOK)
}
