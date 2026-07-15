package routers

import (
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/response"

	"github.com/retail-ai-inc/beanq/v4/helper/json"

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

	data := make(map[string]any)
	count, err := t.client.ZCard(r.Context(), matchStr).Result()
	if err != nil {
		resultRes.Code = berror.InternalServerErrorCode
		resultRes.Msg = err.Error()
		_ = resultRes.Json(w, http.StatusInternalServerError)
		return
	}
	data["total"] = count

	keys, cursor, err := ZScan(r.Context(), t.client, matchStr, gCursor, "*", 10)

	if err != nil {
		resultRes.Code = "1005"
		resultRes.Msg = err.Error()
		_ = resultRes.Json(w, http.StatusInternalServerError)
		return
	}

	msgs := make([]*Msg, 0, len(keys)/2)
	for i := 0; i+1 < len(keys); i += 2 {
		m := new(Msg)
		if err := json.Unmarshal([]byte(keys[i]), m); err != nil {
			continue
		}
		m.Score = keys[i+1]
		msgs = append(msgs, m)
	}

	data["data"] = msgs
	data["cursor"] = cursor
	resultRes.Data = data
	_ = resultRes.Json(w, http.StatusOK)
}
