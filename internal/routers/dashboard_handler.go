package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/spf13/cast"
	"net/http"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

type Dashboard struct {
	client redis.UniversalClient
	prefix string
}

func NewDashboard(client redis.UniversalClient, prefix string) *Dashboard {
	return &Dashboard{client: client, prefix: prefix}
}

func (t *Dashboard) Info(ctx *bwebframework.BeanContext) error {

	result, cancel := response.Get()
	defer cancel()

	w := ctx.Writer
	r := ctx.Request

	server, err := Server(r.Context(), t.client)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusOK)
	}
	persistence, err := Persistence(r.Context(), t.client)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusOK)
	}
	memory, err := Memory(r.Context(), t.client)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusOK)
	}

	command, err := CommandStats(r.Context(), t.client)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusOK)
	}

	clients, err := Clients(r.Context(), t.client)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusOK)
	}
	stats, err := Stats(r.Context(), t.client)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusOK)
	}

	keyspace, err := KeySpace(r.Context(), t.client)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusOK)
	}

	numCpu := runtime.NumCPU()

	// get queue total
	keys, err := Keys(r.Context(), t.client, strings.Join([]string{t.prefix, "*", "stream"}, ":"))
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusInternalServerError)

	}
	keysLen := len(keys)

	// db size
	dbSize, err := DbSize(r.Context(), t.client)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusInternalServerError)
	}

	// Queue Past 10 Minutes
	prefix := viper.GetString("redis.prefix")
	failKey := strings.Join([]string{prefix, "logs", "fail"}, ":")
	failCount := ZCard(r.Context(), t.client, failKey)

	successKey := strings.Join([]string{prefix, "logs", "success"}, ":")
	successCount := ZCard(r.Context(), t.client, successKey)

	queues := make(map[string]any, 0)
	queues["9:20:10"] = map[string]any{"ready": 10, "unacked": 20, "total": 30}
	queues["9:20:20"] = map[string]any{"ready": 5, "unacked": 15, "total": 25}
	queues["9:20:30"] = map[string]any{"ready": 1, "unacked": 10, "total": 20}
	queues["9:20:40"] = map[string]any{"ready": 3, "unacked": 6, "total": 15}
	queues["9:20:50"] = map[string]any{"ready": 4, "unacked": 8, "total": 18}

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
	return result.Json(w, http.StatusOK)
}
