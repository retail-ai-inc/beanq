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
	"github.com/retail-ai-inc/beanq/v3/helper/timex"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/retail-ai-inc/beanq/v3/internal/driver/bredis"
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

	tim := r.URL.Query().Get("time")
	if tim == "" {
		tim = "10"
	}

	// prepare data for charts
	uiTool := bredis.NewUITool(t.client, t.prefix)
	timeDuration := time.Duration(cast.ToInt64(tim)) * time.Second
	go func() {
		timer := timex.TimerPool.Get(timeDuration)
		defer timer.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-timer.C:
				if err := uiTool.QueueMessage(r.Context()); err != nil {
					logger.New().Error(err)
				}
			}
			timer.Reset(timeDuration)
		}
	}()

	nodeId := r.URL.Query().Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	nctx := r.Context()

	timer := timex.TimerPool.Get(300 * time.Millisecond)
	defer timer.Stop()

	var (
		err          error
		keys         []string
		dbSize       int64
		failCount    int64
		successCount int64
	)
	for {
		select {
		case <-nctx.Done():
			return
		case <-timer.C:
		}
		timer.Reset(timeDuration)
		keysLen := 0

		// get queue total
		keys, err = client.Keys(nctx, strings.Join([]string{t.prefix, "*", "stream"}, ":"))
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "dashboard")
			flusher.Flush()
		}
		keysLen = len(keys)

		// db size
		dbSize, err = client.DbSize(nctx)
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "dashboard")
			flusher.Flush()
		}
		// failed count
		failCount, err = t.mog.DocumentCount(nctx, "failed")
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "dashboard")
			flusher.Flush()
		}
		// success count
		successCount, err = t.mog.DocumentCount(nctx, "success")
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "dashboard")
			flusher.Flush()
		}

		//queue messages
		queues := make(map[string]any, 5)
		var qusData = struct {
			TimeKey string `json:"time"`
			Ready   int64  `json:"ready"`
			Unacked int64  `json:"unacked"`
			Total   int64  `json:"total"`
		}{}
		totalkey := strings.Join([]string{t.prefix, "dashboard_total"}, ":")
		qus := t.client.ZRange(nctx, totalkey, 0, -1).Val()
		for _, s := range qus {
			if err := json.NewDecoder(strings.NewReader(s)).Decode(&qusData); err != nil {
				logger.New().Error(err)
				continue
			}
			queues[qusData.TimeKey] = map[string]any{"ready": qusData.Ready, "unacked": qusData.Unacked, "total": qusData.Total}
		}

		// pod status
		hostNameKey := strings.Join([]string{t.prefix, tool.BeanqHostName}, ":")
		pods, _, err := t.client.ZScan(r.Context(), hostNameKey, 0, "*", 10).Result()
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "dashboard")
			flusher.Flush()
			continue
		}

		result.Data = map[string]any{
			"queue_total":   keysLen,
			"db_size":       dbSize,
			"num_cpu":       runtime.NumCPU(),
			"fail_count":    failCount,
			"success_count": successCount,
			"queues":        queues,
			"pods":          pods,
		}
		_ = result.EventMsg(w, "dashboard")
		flusher.Flush()
	}
}
