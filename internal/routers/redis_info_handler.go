package routers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
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
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ctx := r.Context()
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	client := tool.ClientFac(t.client, t.prefix, r.Header.Get("nodeId"))
	eventName := cast.ToString(ctx.Value(EventName{}))
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snapshot, err := client.Snapshot(ctx)
			if err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, eventName)
				flusher.Flush()
				return
			}
			result.Data = map[string]any{
				"info": snapshot.Info, "commands": snapshot.Commands,
				"clients": snapshot.Clients, "stats": snapshot.Stats,
				"keyspace": snapshot.Keyspace, "memory": snapshot.Memory,
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
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	defer flusher.Flush()
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
				res.Code = berror.InternalServerErrorCode
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

	cursor, err := strconv.ParseUint(r.URL.Query().Get("cursor"), 10, 64)
	if r.URL.Query().Get("cursor") == "" {
		cursor = 0
		err = nil
	}
	if err != nil {
		writeBadRequest(w, errors.New("cursor must be an unsigned integer"))
		return
	}
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		pattern = "*"
	}
	keys, next, err := t.client.Scan(r.Context(), cursor, pattern, 100).Result()
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = map[string]any{"data": keys, "nextCursor": next}
	_ = res.Json(w, http.StatusOK)
}

func (t *RedisInfo) DeleteKey(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()
	key := r.PathValue("key")
	if t.prefix != "" && !strings.HasPrefix(key, t.prefix) {
		res.Code = berror.MissParameterCode
		res.Msg = "key is outside the configured prefix"
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	result, err := t.client.Del(r.Context(), key).Result()
	if err != nil {
		res.Code = berror.InternalServerErrorCode
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
	if t.mgo == nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = "mongo is not configured"
		_ = res.Json(w, http.StatusServiceUnavailable)
		return
	}

	var buf bytes.Buffer
	defer r.Body.Close()

	if _, err := io.Copy(&buf, r.Body); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusBadRequest)
		return
	}
	var config capture.Config
	if err := json.Unmarshal(buf.Bytes(), &config); err != nil {
		res.Code = berror.MissParameterCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	if err := t.mgo.AddConfig(r.Context(), &config); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}

	_ = res.Json(w, http.StatusOK)
}

func (t *RedisInfo) ConfigInfo(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()
	if t.mgo == nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = "mongo is not configured"
		_ = res.Json(w, http.StatusServiceUnavailable)
		return
	}

	result, err := t.mgo.ConfigInfo(r.Context())
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = result
	_ = res.Json(w, http.StatusOK)
}
