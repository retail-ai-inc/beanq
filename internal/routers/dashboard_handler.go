package routers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/spf13/cast"
)

type Dashboard struct {
	client redis.UniversalClient
	mog    *bmongo.BMongo
	prefix string
}

// Metrics exposes a real-time Redis snapshot for dashboard advanced metrics.
// Values that Redis cannot calculate are returned as null rather than mocked.
func (t *Dashboard) Metrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	keys, _, err := scanKeys(ctx, t.client, t.prefix+"*:stream*", 0)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, berror.InternalServerErrorCode, err.Error())
		return
	}
	groups := make([]map[string]any, 0)
	delayRows := make([]map[string]any, 0)
	sequenceRows := make([]map[string]any, 0, 20)
	latency := map[string]float64{}
	if t.mog != nil {
		latency, _ = t.mog.EventRuntimePercentiles(ctx, time.Now().Add(-15*time.Minute), 10000)
	}
	var lagTotal int64
	metrics := map[string]int64{}
	for _, name := range []string{"published", "success", "failed"} {
		// Metric counters are stored in minute buckets. Read the newest bucket
		// so requests made after a minute rollover still return the last sample.
		keys, _, scanErr := scanKeys(ctx, t.client, strings.Join([]string{t.prefix, "metrics", name, "*"}, ":"), 0)
		var latestKey string
		var latestBucket int64
		for _, key := range keys {
			bucket, parseErr := strconv.ParseInt(key[strings.LastIndexByte(key, ':')+1:], 10, 64)
			if parseErr == nil && (latestKey == "" || bucket > latestBucket) {
				latestKey, latestBucket = key, bucket
			}
		}
		if scanErr == nil && latestKey != "" {
			if n, getErr := t.client.Get(ctx, latestKey).Int64(); getErr == nil {
				metrics[name] = n
			}
		}
	}
	for _, key := range keys {
		info, e := t.client.XInfoGroups(ctx, key).Result()
		if e != nil {
			continue
		}
		for _, g := range info {
			groups = append(groups, map[string]any{"stream": key, "group": g.Name, "members": g.Consumers, "pending": g.Pending, "lag": g.Lag})
			if g.Lag > 0 {
				lagTotal += g.Lag
			}
		}
	}
	scheduledKeys, _, _ := scanKeys(ctx, t.client, strings.Join([]string{t.prefix, "*", "*", "*", "delay_queue", "scheduled*"}, ":"), 0)
	for _, key := range scheduledKeys {
		parts := strings.Split(key, ":")
		if len(parts) < 6 {
			continue
		}
		total, totalErr := t.client.ZCard(ctx, key).Result()
		due, dueErr := t.client.ZCount(ctx, key, "-inf", cast.ToString(time.Now().UnixMilli())).Result()
		if totalErr != nil || dueErr != nil {
			continue
		}
		delayRows = append(delayRows, map[string]any{"channel": strings.Trim(parts[1], "{}"), "topic": strings.Trim(parts[2], "{}"), "scheduled": total, "dueWaiting": due})
	}
	// Sequence order lists are bounded to the largest 20 lists for dashboard use.
	orderKeys, _, _ := scanKeys(ctx, t.client, strings.Join([]string{t.prefix, "*", "*", "*", "sequence_queue", "*", "order", "*", "list"}, ":"), 0)
	for _, key := range orderKeys {
		length, e := t.client.LLen(ctx, key).Result()
		if e != nil || length == 0 {
			continue
		}
		parts := strings.Split(key, ":")
		if len(parts) < 9 {
			continue
		}
		sequenceRows = append(sequenceRows, map[string]any{"channel": parts[1], "topic": parts[2], "partition": parts[5], "orderKey": parts[7], "listLen": length, "ok": true})
		if len(sequenceRows) >= 20 {
			break
		}
	}
	result, cancel := response.Get()
	defer cancel()
	result.Data = map[string]any{"sampledAt": time.Now().UnixMilli(), "consumerGroups": groups, "lagTotal": lagTotal, "delayQueues": delayRows, "sequenceHotKeys": sequenceRows, "latency": latency, "minute": metrics}
	_ = result.Json(w, http.StatusOK)
}

func NewDashboard(client redis.UniversalClient, x *bmongo.BMongo, prefix string) *Dashboard {
	return &Dashboard{client: client, mog: x, prefix: prefix}
}

func (t *Dashboard) Nodes(w http.ResponseWriter, r *http.Request) {

	nodes := tool.ClientFac(t.client, t.prefix, "").Nodes(r.Context())
	result, cancel := response.Get()
	defer cancel()
	result.Code = berror.SuccessCode
	result.Data = nodes

	_ = result.Json(w, http.StatusOK)
}

func (t *Dashboard) Info(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()
	eventName := cast.ToString(r.Context().Value(EventName{}))

	tim := r.URL.Query().Get("duration")
	if tim == "" {
		tim = "10s"
	}
	tm, err := time.ParseDuration(tim)
	if err != nil || tm < time.Second || tm > 24*time.Hour {
		writeAPIError(w, http.StatusBadRequest, berror.MissParameterCode, "duration must be between 1s and 24h")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	defer flusher.Flush()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()

	totalkey := strings.Join([]string{t.prefix, "dashboard_total"}, ":")
	// Read the latest 1000 change snapshots regardless of their age. Identical
	// idle snapshots are suppressed by the reporter, so duration-based filtering
	// could otherwise return an empty result even though the ZSET has data.
	queues, err := t.client.ZRevRange(ctx, totalkey, 0, 999).Result()
	if err != nil {
		logger.New().Error(err)
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.EventMsg(w, eventName)
		flusher.Flush()
		return
	}
	newQueue := make([]any, 0, len(queues))
	for i := len(queues) - 1; i >= 0; i-- {
		var data any
		if err := json.NewDecoder(strings.NewReader(queues[i])).Decode(&data); err == nil {
			newQueue = append(newQueue, data)
		}
	}
	result.Code = "1111"
	result.Msg = "DONE"
	result.Data = newQueue
	_ = result.EventMsg(w, eventName)
	flusher.Flush()

}

func (t *Dashboard) Total(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	ctx := r.Context()

	nodeId := r.URL.Query().Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	// all keys
	keys, _, err := scanKeys(ctx, t.client, strings.Join([]string{t.prefix, "*", "stream*"}, ":"), 0)
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
	if t.mog == nil {
		writeAPIError(w, http.StatusServiceUnavailable, berror.InternalServerErrorCode, "mongo is not configured")
		return
	}
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

	eventName := cast.ToString(r.Context().Value(EventName{}))

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	defer flusher.Flush()

	hostNameKey := strings.Join([]string{t.prefix, tool.BeanqHostName}, ":")

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			// pod status
			pods, err := t.client.ZRange(r.Context(), hostNameKey, 0, -1).Result()
			if err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, eventName)
				return
			}
			result.Data = pods
			_ = result.EventMsg(w, eventName)
			flusher.Flush()
			ticker.Reset(10 * time.Second)
		}
	}
}
