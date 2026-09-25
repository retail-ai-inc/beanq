package routers

import (
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/bredis"
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
	if err := bredis.PruneHostNameZSet(r.Context(), t.client, hostNameKey, "", time.Now()); err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	pods, err := t.client.ZRange(r.Context(), hostNameKey, 0, -1).Result()
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}

	result.Data = pods
	_ = result.Json(w, http.StatusOK)
}