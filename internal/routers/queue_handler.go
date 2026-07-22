package routers

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
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

	id := r.PathValue("id")
	route := strings.SplitN(id, ":", 2)

	ctx := r.Context()
	emit := func() error {
		stream, err := queueDetailMessages(ctx, client, prefix, route)
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
		} else {
			result.Code = berror.SuccessCode
			result.Msg = ""
			result.Data = stream
		}
		return result.EventMsg(w, "queue_detail")
	}
	if err := streamEvents(w, r, 10*time.Second, emit); err != nil {
		return
	}
}

func queueDetailMessages(ctx context.Context, client redis.UniversalClient, prefix string, route []string) ([]redis.XMessage, error) {
	if len(route) != 2 {
		return nil, nil
	}
	pattern := strings.Join([]string{prefix, route[0], route[1], "*", "normal_queue", "stream*"}, ":")
	keys, _, err := scanKeys(ctx, client, pattern, 0)
	if err != nil {
		return nil, err
	}
	messages := make([]redis.XMessage, 0, 50)
	for _, key := range keys {
		batch, rangeErr := XRangeN(ctx, client, key, "-", "+", 50)
		if rangeErr != nil {
			return nil, rangeErr
		}
		for i := range batch {
			batch[i].Values["partition"] = partitionFromStreamKey(key)
		}
		messages = append(messages, batch...)
	}
	sort.SliceStable(messages, func(i, j int) bool { return compareStreamID(messages[i].ID, messages[j].ID) < 0 })
	if len(messages) > 50 {
		messages = messages[:50]
	}
	return messages, nil
}

func partitionFromStreamKey(key string) string {
	parts := strings.Split(key, ":")
	for i, part := range parts {
		if (part == "normal_queue" || part == "delay_queue" || part == "sequence_queue") && i+1 < len(parts) {
			return strings.TrimPrefix(parts[i+1], "stream-")
		}
	}
	return "legacy"
}

func compareStreamID(left, right string) int {
	parse := func(id string) (int64, int64) {
		parts := strings.SplitN(id, "-", 2)
		ms, _ := strconv.ParseInt(parts[0], 10, 64)
		var seq int64
		if len(parts) == 2 {
			seq, _ = strconv.ParseInt(parts[1], 10, 64)
		}
		return ms, seq
	}
	lm, ls := parse(left)
	rm, rs := parse(right)
	if lm < rm || (lm == rm && ls < rs) {
		return -1
	}
	if lm == rm && ls == rs {
		return 0
	}
	return 1
}
