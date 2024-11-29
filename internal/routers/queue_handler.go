package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Queue struct {
	client redis.UniversalClient
	prefix string
}

func NewQueue(client redis.UniversalClient, prefix string) *Queue {
	return &Queue{client: client, prefix: prefix}
}

func (t *Queue) List(ctx *bwebframework.BeanContext) error {
	result, cancel := response.Get()
	defer cancel()

	bt, err := QueueInfo(ctx.Request.Context(), t.client, t.prefix)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(ctx.Writer, http.StatusInternalServerError)
	}

	result.Data = bt
	return result.Json(ctx.Writer, http.StatusOK)

}
func (t *Queue) Detail(ctx *bwebframework.BeanContext) error {
	queueDetail(ctx.Writer, ctx.Request, t.client)
	return nil
}

func queueDetail(w http.ResponseWriter, r *http.Request, client redis.UniversalClient) {

	result, cancel := response.Get()
	defer cancel()

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	id := r.FormValue("id")
	prefix := viper.GetString("redis.prefix")
	id = strings.Join([]string{prefix, id, "normal_stream", "stream"}, ":")

	ctx := r.Context()
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stream, err := XRangeN(ctx, client, id, "-", "+", 50)

			if err != nil {
				result.Code = "1004"
				result.Msg = err.Error()
			}

			if err == nil {
				result.Data = stream
			}
			_ = result.EventMsg(w, "queue_detail")
			flusher.Flush()
			ticker.Reset(10 * time.Second)
		}
	}
}
