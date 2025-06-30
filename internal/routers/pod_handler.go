package routers

import (
	"net/http"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
)

type Pod struct {
	client redis.UniversalClient
	mog    *bmongo.BMongo
	prefix string
}

func NewPod(client redis.UniversalClient, mongo *bmongo.BMongo, prefix string) *Pod {

	return &Pod{
		client: client,
		mog:    mongo,
		prefix: prefix,
	}
}

func (t *Pod) List(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	hostNameKey := strings.Join([]string{t.prefix, tool.BeanqHostName}, ":")
	cmd := t.client.HGetAll(r.Context(), hostNameKey)
	if cmd.Err() != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = cmd.Err().Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}

	result.Data = cmd.Val()
	_ = result.Json(w, http.StatusOK)
}
