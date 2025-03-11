package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
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

func (t *Logs) List(w http.ResponseWriter, r *http.Request) {

	resultRes, cancel := response.Get()
	defer cancel()

	var (
		dataType string
		matchStr = strings.Join([]string{t.prefix, "logs", "success"}, ":")
	)

	dataType = r.FormValue("type")
	gCursor := cast.ToUint64(r.FormValue("cursor"))

	if dataType != "success" && dataType != "error" {
		resultRes.Code = berror.TypeErrorCode
		resultRes.Msg = berror.TypeErrorMsg
		_ = resultRes.Json(w, http.StatusInternalServerError)
		return
	}

	if dataType == "error" {
		matchStr = strings.Join([]string{t.prefix, "logs", "fail"}, ":")
	}

	nodeId := r.Header.Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	data := make(map[string]any)
	count, err := client.ZCard(r.Context(), matchStr)
	if err != nil {
		resultRes.Code = berror.InternalServerErrorCode
		resultRes.Msg = err.Error()
		_ = resultRes.Json(w, http.StatusInternalServerError)
		return
	}
	data["total"] = count

	keys, cursor, err := ZScan(r.Context(), t.client, matchStr, gCursor, "", 10)

	if err != nil {
		resultRes.Code = "1005"
		resultRes.Msg = err.Error()
		_ = resultRes.Json(w, http.StatusInternalServerError)
		return
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
	_ = resultRes.Json(w, http.StatusOK)
}
