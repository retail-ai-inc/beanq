package routers

import (
	"net/http"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"go.mongodb.org/mongo-driver/bson"
)

type Dlq struct {
	client redis.UniversalClient
	prefix string
	mgo    *bmongo.BMongo
}

func NewDlq(client redis.UniversalClient, mongo *bmongo.BMongo, prefix string) *Dlq {
	return &Dlq{client: client, mgo: mongo, prefix: prefix}
}

func (t *Dlq) List(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	page, err := parsePage(r)
	if err != nil {
		writeBadRequest(w, err)
		return
	}
	query := r.URL.Query()
	id := query.Get("id")
	status := query.Get("status")
	moodType := query.Get("moodType")
	topicName := query.Get("topicName")

	filter := bson.M{}
	filter["logType"] = bstatus.Dlq
	if id != "" {
		filter["id"] = id
	}
	if status != "" {
		filter["status"] = status
	}
	if moodType != "" {
		filter["moodType"] = moodType
	}
	if topicName != "" {
		filter["topic"] = topicName
	}
	datas := make(map[string]any, 3)
	data, total, err := t.mgo.EventLogs(r.Context(), filter, page.Page, page.PageSize)
	if err != nil {
		result.Code = "1001"
		result.Msg = err.Error()
	}
	if err == nil {
		datas["data"] = data
		datas["total"] = total
		datas["cursor"] = page.Page
		result.Data = datas
	}
	_ = result.Json(w, http.StatusOK)
}

func (t *Dlq) Delete(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	id := r.PostFormValue("id")
	count, err := t.mgo.Delete(r.Context(), id)
	if err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	if count == 0 {
		writeAPIError(w, http.StatusNotFound, berror.MissParameterCode, "dead letter not found")
		return
	}
	res.Data = count
	_ = res.Json(w, http.StatusOK)
}

func (t *Dlq) Retry(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	id := r.FormValue("uniqueId")

	nctx := r.Context()

	data := make(map[string]any)
	if err := json.Unmarshal([]byte(r.FormValue("data")), &data); err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}

	if err := publishRetry(nctx, data, defaultRetryPublisherFactory(t.client, t.prefix)); err != nil {
		res.Msg = err.Error()
		res.Code = berror.TypeErrorCode
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	if _, err := t.mgo.Delete(nctx, id); err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	_ = res.Json(w, http.StatusOK)
}
