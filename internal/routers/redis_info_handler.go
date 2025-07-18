package routers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/spf13/cast"
)

type RedisInfo struct {
	client redis.UniversalClient
	prefix string
	mgo    *bmongo.BMongo
}

func NewRedisInfo(client redis.UniversalClient, prefix string, mongo *bmongo.BMongo) *RedisInfo {
	return &RedisInfo{client: client, prefix: prefix, mgo: mongo}
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
		memory    map[string]any
		command   []map[string]any
		clients   map[string]any
		stats     map[string]any
		keyspace  []map[string]any
		eventName = cast.ToString(r.Context().Value(EventName{}))
	)

	for {
		select {
		case <-nctx.Done():
			return
		case <-ticker.C:
			d, err := client.Info(nctx)

			memory, err = client.Memory(nctx)
			if err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, eventName)
				flusher.Flush()
				return
			}

			command, err = client.CommandStats(nctx)
			if err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, eventName)
				flusher.Flush()
				return
			}

			clients, err = client.Clients(nctx)
			if err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, eventName)
				flusher.Flush()
				return
			}

			stats, err = client.Stats(nctx)
			if err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, eventName)
				flusher.Flush()
				return
			}

			keyspace, err = client.KeySpace(nctx)
			if err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, eventName)
				flusher.Flush()
				return
			}

			result.Data = map[string]any{
				"info":     d,
				"commands": command,
				"clients":  clients,
				"stats":    stats,
				"keyspace": keyspace,
				"memory":   memory,
			}

			_ = result.EventMsg(w, eventName)
			flusher.Flush()
			ticker.Reset(10 * time.Second)

		}
	}
}

func (t *RedisInfo) Monitor(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	eventName := cast.ToString(r.Context().Value(EventName{}))

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
				_ = res.EventMsg(w, eventName)
				return
			}
			str = strings.ReplaceAll(str, "MONITOR:", "")
			if strings.Contains(str, "OK") {
				continue
			}
			res.Data = fmt.Sprintf("Time:%s,Command:%s", time.Now(), str)
			_ = res.EventMsg(w, eventName)
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

	var buf bytes.Buffer
	defer r.Body.Close()

	if _, err := io.Copy(&buf, r.Body); err != nil {
		res.Code = response.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusBadRequest)
		return
	}
	var config capture.Config
	if err := json.Unmarshal(buf.Bytes(), &config); err != nil {
		res.Code = response.MissParameterCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	if err := t.mgo.AddConfig(r.Context(), &config); err != nil {
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

	result, err := t.mgo.ConfigInfo(r.Context())
	if err != nil {
		res.Code = response.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = result
	_ = res.Json(w, http.StatusOK)
}
