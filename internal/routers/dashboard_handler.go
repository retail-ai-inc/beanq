package routers

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/timex"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"net/http"
	"runtime"
	"strings"
	"time"
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
		timer.Reset(10 * time.Second)

		numCpu := runtime.NumCPU()

		func() {
			ctx8, cancel8 := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel8()
			// get queue total
			keys, err = client.Keys(ctx8, strings.Join([]string{t.prefix, "*", "stream"}, ":"))
			if err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, "dashboard")
				flusher.Flush()
			}
		}()

		keysLen := len(keys)

		func() {
			ctx9, cancel9 := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel9()
			// db size
			dbSize, err = client.DbSize(ctx9)
			if err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, "dashboard")
				flusher.Flush()
			}
		}()

		func() {
			ctx10, cancel10 := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel10()
			failCount, err = t.mog.DocumentCount(ctx10, "failed")
			if err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, "dashboard")
				flusher.Flush()
			}
		}()
		func() {
			ctx11, cancel11 := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel11()
			successCount, err = t.mog.DocumentCount(ctx11, "success")
			if err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, "dashboard")
				flusher.Flush()
			}
		}()

		//queue messages
		queues := make(map[string]any, 0)
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

		pods, _, err := t.client.ZScan(r.Context(), tool.BeanqHostName, 0, "*", 10).Result()
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "dashboard")
			flusher.Flush()
		}

		result.Data = map[string]any{
			"queue_total":   keysLen,
			"db_size":       dbSize,
			"num_cpu":       numCpu,
			"fail_count":    failCount,
			"success_count": successCount,
			"queues":        queues,
			"pods":          pods,
		}
		_ = result.EventMsg(w, "dashboard")
		flusher.Flush()
	}
}
