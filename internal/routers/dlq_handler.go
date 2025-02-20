package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	public "github.com/retail-ai-inc/beanq/v3/internal"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/retail-ai-inc/beanq/v3/internal/driver/bredis"
	"go.mongodb.org/mongo-driver/bson"
	"time"

	"github.com/spf13/cast"
	"net/http"
)

type Dlq struct {
	client redis.UniversalClient
	prefix string
	mgo    *bmongo.BMongo
}

func NewDlq(client redis.UniversalClient, mongo *bmongo.BMongo, prefix string) *Dlq {
	return &Dlq{client: client, mgo: mongo, prefix: prefix}
}

func (t *Dlq) List(ctx *bwebframework.BeanContext) error {

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
	filter["logType"] = bstatus.Dlq

	datas := make(map[string]any, 3)
	data, total, err := t.mgo.EventLogs(r.Context(), filter, page, pageSize)
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

func (t *Dlq) Delete(ctx *bwebframework.BeanContext) error {

	res, cancel := response.Get()
	defer cancel()

	w := ctx.Writer
	r := ctx.Request

	id := r.PostFormValue("id")
	count, err := t.mgo.Delete(r.Context(), id)
	if err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		return res.Json(w, http.StatusInternalServerError)
	}
	res.Data = count
	return res.Json(w, http.StatusOK)
}

func (t *Dlq) Retry(ctx *bwebframework.BeanContext) error {

	res, cancel := response.Get()
	defer cancel()

	w := ctx.Writer
	r := ctx.Request

	m := make(map[string]any)
	id := r.FormValue("id")
	m["uniqueId"] = id
	nctx := r.Context()

	data := make(map[string]any)
	if err := json.Unmarshal([]byte(r.FormValue("data")), &data); err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		return res.Json(w, http.StatusInternalServerError)
	}

	moodType := ""
	if v, ok := data["moodType"]; ok {
		moodType = v.(string)
	}

	var bk public.IBroker
	if moodType == string(btype.SEQUENTIAL) {
		return res.Json(w, http.StatusOK)
	}
	if moodType == string(btype.DELAY) {

		bk = bredis.NewSchedule(t.client, t.prefix, 10, 20*time.Minute)
		if err := bk.Enqueue(nctx, data); err != nil {
			res.Msg = err.Error()
			res.Code = berror.InternalServerErrorCode
			return res.Json(w, http.StatusOK)
		}
		return res.Json(w, http.StatusOK)
	}

	bk = bredis.NewNormal(t.client, t.prefix, 2000, 10, 20)
	if err := bk.Enqueue(nctx, data); err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		return res.Json(w, http.StatusOK)
	}

	return res.Json(w, http.StatusOK)
}
