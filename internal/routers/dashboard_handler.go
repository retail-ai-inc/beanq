package routers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/spf13/cast"
)

type Dashboard struct {
	client redis.UniversalClient
	mog    *bmongo.BMongo
	prefix string
}

func NewDashboard(client redis.UniversalClient, x *bmongo.BMongo, prefix string) *Dashboard {
	return &Dashboard{client: client, mog: x, prefix: prefix}
}

func (t *Dashboard) Nodes(w http.ResponseWriter, r *http.Request) {

	nodes := tool.ClientFac(t.client, t.prefix, "").Nodes(r.Context())
	result, cancel := response.Get()
	defer cancel()
	result.Code = response.SuccessCode
	result.Data = nodes

	_ = result.Json(w, http.StatusOK)
}

func (t *Dashboard) Info(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	tm := 5 * time.Second
	tim := r.URL.Query().Get("time")
	if tim == "" {
		tim = "10s"
	}
	if strings.Contains(tim, "s") {
		tim = strings.ReplaceAll(tim, "s", "")
		tm = time.Duration(cast.ToInt64(tim)) * time.Second
	}
	if strings.Contains(tim, "m") {
		tim = strings.ReplaceAll(tim, "m", "")
		tm = time.Duration(cast.ToInt64(tim)) * 60 * time.Second
	}
	if strings.Contains(tim, "h") {
		tim = strings.ReplaceAll(tim, "h", "")
		tm = time.Duration(cast.ToInt64(tim)) * 60 * 60 * time.Second
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()

	nodeId := r.URL.Query().Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	totalkey := strings.Join([]string{t.prefix, "dashboard_total"}, ":")
	now := time.Now()
	before := now.Add(-tm)
	beforeStr := cast.ToString(before.Unix())
	nowStr := cast.ToString(now.Unix())

	count := int64(1000)
	offset := int64(0)

	zcount := client.ZCount(ctx, totalkey, beforeStr, nowStr)
	page := zcount / count

	newQueue := make([]map[string]any, 0, zcount)

	for {
		if page < 0 {
			break
		}
		queues, err := client.ZRangeByScore(ctx, totalkey, beforeStr, nowStr, offset, count)
		if err != nil {
			logger.New().Error(err)
			continue
		}

		for _, queue := range queues {
			m := make(map[string]any, 0)
			if err := json.NewDecoder(strings.NewReader(queue)).Decode(&m); err != nil {
				continue
			}
			newQueue = append(newQueue, m)
		}
		offset += count
		page--
		result.Data = newQueue
		_ = result.EventMsg(w, "dashboard")
		flusher.Flush()
		newQueue = newQueue[:0]
	}
	result.Code = "1111"
	result.Data = "DONE"
	_ = result.EventMsg(w, "dashboard")
	flusher.Flush()
	//return
}

func (t *Dashboard) Total(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	ctx := r.Context()

	nodeId := r.URL.Query().Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	// all keys
	keys, err := client.Keys(ctx, strings.Join([]string{t.prefix, "*", "stream"}, ":"))
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}

	// db size
	dbSize, err := client.DbSize(ctx)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}

	// failed count
	failCount, err := t.mog.DocumentCount(ctx, "failed")
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	// success count
	successCount, err := t.mog.DocumentCount(ctx, "success")
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	result.Data = map[string]any{
		"queue_total":   len(keys),
		"db_size":       dbSize,
		"num_cpu":       runtime.NumCPU(),
		"fail_count":    failCount,
		"success_count": successCount,
	}
	_ = result.Json(w, http.StatusOK)
}

func (t *Dashboard) Pods(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	// pod status
	hostNameKey := strings.Join([]string{t.prefix, tool.BeanqHostName}, ":")
	pods, err := t.client.ZRange(r.Context(), hostNameKey, 0, -1).Result()
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	result.Data = pods
	_ = result.Json(w, http.StatusOK)
}
