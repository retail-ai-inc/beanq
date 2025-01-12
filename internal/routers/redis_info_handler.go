package routers

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"net/http"
	"strings"
	"time"
)

type RedisInfo struct {
	client redis.UniversalClient
	prefix string
}

func NewRedisInfo(client redis.UniversalClient, prefix string) *RedisInfo {
	return &RedisInfo{client: client, prefix: prefix}
}
func (t *RedisInfo) Info(ctx *bwebframework.BeanContext) error {

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

	nodeId := r.Header.Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	for {
		select {
		case <-nctx.Done():
			return nctx.Err()
		case <-ticker.C:
			d, err := client.Info(nctx)

			if err != nil {
				result.Code = "1001"
				result.Msg = err.Error()
			}

			if err == nil {
				result.Data = d
			}
			_ = result.EventMsg(w, "redis_info")
			flusher.Flush()
			ticker.Reset(10 * time.Second)

		}
	}
}

func (t *RedisInfo) Monitor(ctx *bwebframework.BeanContext) error {

	res, cancel := response.Get()
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	w := ctx.Writer
	r := ctx.Request

	flusher, ok := w.(http.Flusher)
	defer flusher.Flush()
	if !ok {
		http.Error(w, "server error", http.StatusInternalServerError)
		return nil
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	nodeId := r.Header.Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	for {
		select {
		case <-r.Context().Done():
			return nil
		case <-ticker.C:

			str, err := client.Monitor(r.Context())
			if err != nil {
				res.Code = response.InternalServerErrorCode
				res.Msg = err.Error()
				return res.EventMsg(w, "redis_monitor")
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
