package routers

import (
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/retail-ai-inc/beanq/v3/helper/mongox"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"net/http"
	"runtime"
	"strings"
	"time"
)

type Dashboard struct {
	client redis.UniversalClient
	mog    *mongox.MongoX
	prefix string
}

func NewDashboard(client redis.UniversalClient, x *mongox.MongoX, prefix string) *Dashboard {
	return &Dashboard{client: client, mog: x, prefix: prefix}
}

func (t *Dashboard) Info(ctx *bwebframework.BeanContext) error {

	result, cancel := response.Get()
	defer cancel()

	w := ctx.Writer
	r := ctx.Request

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "server error", http.StatusInternalServerError)
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	nctx := r.Context()
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-nctx.Done():
			return nctx.Err()
		default:

		}
		ticker.Reset(10 * time.Second)
		server, err := Server(r.Context(), t.client)
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "redis_info")
			flusher.Flush()
			return nil
		}
		persistence, err := Persistence(r.Context(), t.client)
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "redis_info")
			flusher.Flush()
			return nil
		}
		memory, err := Memory(r.Context(), t.client)
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "redis_info")
			flusher.Flush()
			return nil
		}

		command, err := CommandStats(r.Context(), t.client)
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "redis_info")
			flusher.Flush()
			return nil
		}

		clients, err := Clients(r.Context(), t.client)
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "redis_info")
			flusher.Flush()
			return nil
		}
		stats, err := Stats(r.Context(), t.client)
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "redis_info")
			flusher.Flush()
			return nil
		}

		keyspace, err := KeySpace(r.Context(), t.client)
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "redis_info")
			flusher.Flush()
			return nil
		}

		numCpu := runtime.NumCPU()

		// get queue total
		keys, err := Keys(r.Context(), t.client, strings.Join([]string{t.prefix, "*", "stream"}, ":"))
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "redis_info")
			flusher.Flush()
			return nil
		}
		keysLen := len(keys)

		// db size
		dbSize, err := DbSize(r.Context(), t.client)
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, "redis_info")
			flusher.Flush()
			return nil
		}

		// Queue Past 10 Minutes
		prefix := viper.GetString("redis.prefix")
		failKey := strings.Join([]string{prefix, "logs", "fail"}, ":")
		failCount := ZCard(r.Context(), t.client, failKey)

		successKey := strings.Join([]string{prefix, "logs", "success"}, ":")
		successCount := ZCard(r.Context(), t.client, successKey)

		//queue messages
		queues := make(map[string]any, 0)
		var qusData = struct {
			Ready   int64  `json:"ready"`
			Unacked int64  `json:"unacked"`
			Total   int64  `json:"total"`
			TimeKey string `json:"time"`
		}{}
		totalkey := strings.Join([]string{t.prefix, "dashboard_total"}, ":")
		qus := t.client.ZRange(ctx.Request.Context(), totalkey, 0, -1).Val()
		for _, s := range qus {
			if err := json.NewDecoder(strings.NewReader(s)).Decode(&qusData); err != nil {
				logger.New().Error(err)
				continue
			}
			queues[qusData.TimeKey] = map[string]any{"ready": qusData.Ready, "unacked": qusData.Unacked, "total": qusData.Total}
		}

		result.Data = map[string]any{
			"queue_total":   keysLen,
			"db_size":       dbSize,
			"num_cpu":       numCpu,
			"fail_count":    failCount,
			"success_count": successCount,
			"used_memory":   cast.ToInt(memory["used_memory_rss"]) / 1024 / 1024,
			"total_memory":  cast.ToInt(memory["total_system_memory"]) / 1024 / 1024,
			"commands":      command,
			"clients":       clients,
			"stats":         stats,
			"keyspace":      keyspace,
			"memory":        memory,
			"server":        server,
			"persistence":   persistence,
			"queues":        queues,
		}
		_ = result.EventMsg(w, "dashboard")
		flusher.Flush()
	}
	return nil
}
