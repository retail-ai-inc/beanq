package routers

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
)

type Queue struct {
	client redis.UniversalClient
	prefix string
}

func NewQueue(client redis.UniversalClient, prefix string) *Queue {
	return &Queue{client: client, prefix: prefix}
}

func (t *Queue) List(w http.ResponseWriter, r *http.Request) {
	result, cancel := response.Get()
	defer cancel()

	bt, err := QueueInfo(r.Context(), t.client, t.prefix)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}

	result.Data = bt
	_ = result.Json(w, http.StatusOK)

}
func (t *Queue) Detail(w http.ResponseWriter, r *http.Request) {
	queueDetail(w, r, t.client, t.prefix)
}

func queueDetail(w http.ResponseWriter, r *http.Request, client redis.UniversalClient, prefix string) {

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
