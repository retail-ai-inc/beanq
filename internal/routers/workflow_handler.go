package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

type WorkFlow struct {
	client redis.UniversalClient
	prefix string
	mgo    *bmongo.BMongo
}

func NewWorkFlow(client redis.UniversalClient, mongo *bmongo.BMongo, prefix string) *WorkFlow {
	return &WorkFlow{client: client, mgo: mongo, prefix: prefix}
}

func (t *WorkFlow) List(ctx *bwebframework.BeanContext) error {

	w := ctx.Writer
	r := ctx.Request

	result, cancel := response.Get()
	defer cancel()

	query := r.URL.Query()
	page := cast.ToInt64(query.Get("page"))
	pageSize := cast.ToInt64(query.Get("pageSize"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	filter := bson.M{}

	datas := make(map[string]any, 3)
	data, total, err := t.mgo.WorkFLowLogs(r.Context(), filter, page, pageSize)
	if err != nil {
		result.Code = "1001"
		result.Msg = err.Error()
	}
	if err == nil {
		datas["data"] = data
		datas["total"] = total
		datas["cursor"] = page
		result.Data = datas
	}
	return result.Json(w, http.StatusOK)

}

func (t *WorkFlow) Delete(ctx *bwebframework.BeanContext) error {

	res, cancel := response.Get()
	defer cancel()

	w := ctx.Writer
	r := ctx.Request

	id := r.PostFormValue("id")
	count, err := t.mgo.DeleteWorkFlow(r.Context(), id)
	if err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		return res.Json(w, http.StatusInternalServerError)
	}
	res.Data = count
	return res.Json(w, http.StatusOK)
}
