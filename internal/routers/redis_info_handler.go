package routers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/retail-ai-inc/beanq/v3/internal/capture"
)

type RedisInfo struct {
	client redis.UniversalClient
	prefix string
}

func NewRedisInfo(client redis.UniversalClient, prefix string) *RedisInfo {
	return &RedisInfo{client: client, prefix: prefix}
}
func (t *RedisInfo) Info(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "server error", http.StatusInternalServerError)
		flusher.Flush()
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	nctx := r.Context()
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	nodeId := r.Header.Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	var (
		//redis info
		memory   map[string]any
		command  []map[string]any
		clients  map[string]any
		stats    map[string]any
		keyspace []map[string]any
	)

	for {
		select {
		case <-nctx.Done():
			return
		case <-ticker.C:
			d, err := client.Info(nctx)

			func() {
				ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel3()
				memory, err = client.Memory(ctx3)
				if err != nil {
					result.Code = berror.InternalServerErrorCode
					result.Msg = err.Error()
					_ = result.EventMsg(w, "dashboard")
					flusher.Flush()
				}
			}()
			func() {
				ctx4, cancel4 := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel4()
				command, err = client.CommandStats(ctx4)
				if err != nil {
					result.Code = berror.InternalServerErrorCode
					result.Msg = err.Error()
					_ = result.EventMsg(w, "dashboard")
					flusher.Flush()
				}
			}()

			func() {
				ctx5, cancel5 := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel5()
				clients, err = client.Clients(ctx5)
				if err != nil {
					result.Code = berror.InternalServerErrorCode
					result.Msg = err.Error()
					_ = result.EventMsg(w, "dashboard")
					flusher.Flush()
				}
			}()
			func() {
				ctx6, cancel6 := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel6()
				stats, err = client.Stats(ctx6)
				if err != nil {
					result.Code = berror.InternalServerErrorCode
					result.Msg = err.Error()
					_ = result.EventMsg(w, "dashboard")
					flusher.Flush()
				}
			}()
			func() {
				ctx7, cancel7 := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel7()
				keyspace, err = client.KeySpace(ctx7)
				if err != nil {
					result.Code = berror.InternalServerErrorCode
					result.Msg = err.Error()
					_ = result.EventMsg(w, "dashboard")
					flusher.Flush()
				}
			}()

			if err != nil {
				result.Code = "1001"
				result.Msg = err.Error()
			}

			result.Data = map[string]any{
				"info":     d,
				"commands": command,
				"clients":  clients,
				"stats":    stats,
				"keyspace": keyspace,
				"memory":   memory,
			}

			_ = result.EventMsg(w, "redis_info")
			flusher.Flush()
			ticker.Reset(10 * time.Second)

		}
	}
}

func (t *RedisInfo) Monitor(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	flusher, ok := w.(http.Flusher)
	defer flusher.Flush()
	if !ok {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	nodeId := r.Header.Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:

			str, err := client.Monitor(r.Context())
			if err != nil {
				res.Code = response.InternalServerErrorCode
				res.Msg = err.Error()
				_ = res.EventMsg(w, "redis_monitor")
				return
			}
			str = strings.ReplaceAll(str, "MONITOR:", "")
			if strings.Contains(str, "OK") {
				continue
			}
			res.Data = fmt.Sprintf("Time:%s,Command:%s", time.Now(), str)
			_ = res.EventMsg(w, "redis_monitor")
			flusher.Flush()
			ticker.Reset(time.Second)
		}
	}
}

func (t *RedisInfo) Keys(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	result, err := t.client.Keys(r.Context(), "*").Result()
	if err != nil {
		res.Code = response.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = result
	_ = res.Json(w, http.StatusOK)
}

func (t *RedisInfo) DeleteKey(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()
	key := r.PathValue("key")

	result, err := t.client.Del(r.Context(), key).Result()
	if err != nil {
		res.Code = response.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = result
	_ = res.Json(w, http.StatusOK)
}

func (t *RedisInfo) Config(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	buf := bytes.NewBuffer(nil)
	defer r.Body.Close()

	if _, err := io.Copy(buf, r.Body); err != nil {
		res.Code = response.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	var NewConfig struct {
		Data string `json:"data"`
	}
	decode := json.NewDecoder(buf)
	if err := decode.Decode(&NewConfig); err != nil {
		res.Code = response.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	var config capture.Config

	if err := json.Unmarshal([]byte(NewConfig.Data), &config); err != nil {
		res.Code = response.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	data := make(map[string]any, 4)

	data["google"] = config.Google
	data["smtp"] = config.SMTP
	data["sendGrid"] = config.SendGrid
	data["rule"] = config.Rule
	data["slack"] = config.Slack
	if err := t.client.HSet(r.Context(), strings.Join([]string{t.prefix, "config"}, ":"), data).Err(); err != nil {
		res.Code = response.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	_ = res.Json(w, http.StatusOK)
}

func (t *RedisInfo) ConfigInfo(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	data, err := t.client.HGetAll(r.Context(), strings.Join([]string{t.prefix, "config"}, ":")).Result()
	if err != nil {
		res.Code = response.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = data
	_ = res.Json(w, http.StatusOK)
}
