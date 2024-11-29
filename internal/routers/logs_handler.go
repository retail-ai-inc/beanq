package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"net/http"
	"strings"

	"github.com/retail-ai-inc/beanq/v3/helper/json"

	"github.com/spf13/cast"
)

type Logs struct {
	client redis.UniversalClient
	prefix string
}

func NewLogs(client redis.UniversalClient, prefix string) *Logs {
	return &Logs{client: client, prefix: prefix}
}

func (t *Logs) List(ctx *bwebframework.BeanContext) error {

	resultRes, cancel := response.Get()
	defer cancel()

	var (
		dataType string
		matchStr = strings.Join([]string{t.prefix, "logs", "success"}, ":")
	)
	w := ctx.Writer
	r := ctx.Request

	dataType = r.FormValue("type")
	gCursor := cast.ToUint64(r.FormValue("cursor"))

	if dataType != "success" && dataType != "error" {
		resultRes.Code = berror.TypeErrorCode
		resultRes.Msg = berror.TypeErrorMsg

		return resultRes.Json(w, http.StatusInternalServerError)
	}

	if dataType == "error" {
		matchStr = strings.Join([]string{t.prefix, "logs", "fail"}, ":")
	}
	data := make(map[string]any)
	count := ZCard(r.Context(), t.client, matchStr)
	data["total"] = count

	keys, cursor, err := ZScan(r.Context(), t.client, matchStr, gCursor, "", 10)

	if err != nil {
		resultRes.Code = "1005"
		resultRes.Msg = err.Error()
		return resultRes.Json(w, http.StatusInternalServerError)
	}

	msgs := make([]*Msg, 0, 10)
	m := new(Msg)

	for _, key := range keys {

		if err := json.Unmarshal([]byte(key), &m); err != nil {
			m.Score = key
			msgs = append(msgs, m)
			m = nil
		}

	}

	data["data"] = msgs
	data["cursor"] = cursor
	resultRes.Data = data

	return resultRes.Json(w, http.StatusOK)

}
