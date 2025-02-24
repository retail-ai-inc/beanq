package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
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

func (t *Pod) List(beanContext *bwebframework.BeanContext) error {

	result, cancel := response.Get()
	defer cancel()

	w := beanContext.Writer
	r := beanContext.Request

	cmd := t.client.SMembers(r.Context(), tool.BeanqHostName)
	vals, err := cmd.Result()
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusInternalServerError)
	}

	var data = make(map[string][]bson.M)
	for _, val := range vals {
		res, err := t.mog.LogsByPod(r.Context(), val)
		if err != nil {
			continue
		}
		data[val] = res
	}
	result.Data = data
	return result.Json(w, http.StatusOK)
}
