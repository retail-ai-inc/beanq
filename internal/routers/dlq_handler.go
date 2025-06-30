package routers

import (
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/bredis"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spf13/cast"
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

	query := r.URL.Query()
	page := cast.ToInt64(query.Get("page"))
	pageSize := cast.ToInt64(query.Get("pageSize"))
	id := query.Get("id")
	status := query.Get("status")
	moodType := query.Get("moodType")
	topicName := query.Get("topicName")

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
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

	moodType := ""
	if v, ok := data["moodType"]; ok {
		moodType = v.(string)
	}

	var bk public.IBroker
	if moodType == string(btype.SEQUENCE) {
		_ = res.Json(w, http.StatusOK)
		return
	}
	if moodType == string(btype.DELAY) {

		bk = bredis.NewSchedule(t.client, t.prefix, 100, 10, 20*time.Minute, nil)
		if err := bk.Enqueue(nctx, data); err != nil {
			res.Msg = err.Error()
			res.Code = berror.InternalServerErrorCode
			_ = res.Json(w, http.StatusInternalServerError)
			return
		}
		_ = res.Json(w, http.StatusOK)
		return
	}

	bk = bredis.NewNormal(t.client, t.prefix, 2000, 100, 10, 20, nil)
	if err := bk.Enqueue(nctx, data); err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		_ = res.Json(w, http.StatusInternalServerError)
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
