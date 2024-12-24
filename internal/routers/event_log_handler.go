package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/retail-ai-inc/beanq/v3/helper/mongox"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	public "github.com/retail-ai-inc/beanq/v3/internal"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/retail-ai-inc/beanq/v3/internal/driver/bredis"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"
)

type EventLog struct {
	Id     string `json:"id"`
	client redis.UniversalClient
	mogx   *mongox.MongoX
	prefix string
}

func NewEventLog(client redis.UniversalClient, x *mongox.MongoX, prefix string) *EventLog {
	return &EventLog{client: client, mogx: x, prefix: prefix}
}

func (t *EventLog) List(ctx *bwebframework.BeanContext) error {

	r := ctx.Request
	w := ctx.Writer

	result, cancel := response.Get()
	defer func() {
		cancel()
	}()
	query := r.URL.Query()
	page := cast.ToInt64(query.Get("page"))
	pageSize := cast.ToInt64(query.Get("pageSize"))
	id := query.Get("id")
	status := query.Get("status")

	filter := bson.M{}
	filter["logType"] = bstatus.Logic
	if id != "" {
		if _, err := primitive.ObjectIDFromHex(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return nil
		}
		filter["id"] = id
	}
	if status != "" {
		filter["status"] = status
	}
	if page <= 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	flush, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "server err", http.StatusInternalServerError)
		return nil
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	datas := make(map[string]any, 3)
	nctx := r.Context()

	for {
		select {
		case <-nctx.Done():
			return nctx.Err()
		case <-ticker.C:

			data, total, err := t.mogx.EventLogs(nctx, filter, page, pageSize)
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

			_ = result.EventMsg(w, "event_log")
			flush.Flush()
			ticker.Reset(5 * time.Second)
		}
	}
}

func (t *EventLog) Detail(ctx *bwebframework.BeanContext) error {

	res, cancel := response.Get()
	defer cancel()

	r := ctx.Request
	w := ctx.Writer

	id := r.URL.Query().Get("id")
	data, err := t.mogx.DetailEventLog(r.Context(), id)
	if err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		return res.Json(w, http.StatusInternalServerError)

	}
	res.Data = data
	return res.Json(w, http.StatusOK)

}

func (t *EventLog) Delete(ctx *bwebframework.BeanContext) error {

	res, cancel := response.Get()
	defer cancel()

	w := ctx.Writer
	r := ctx.Request

	id := r.PostFormValue("id")
	count, err := t.mogx.Delete(r.Context(), id)
	if err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		return res.Json(w, http.StatusInternalServerError)
	}
	res.Data = count
	return res.Json(w, http.StatusOK)
}

func (t *EventLog) Edit(ctx *bwebframework.BeanContext) error {
	res, cancel := response.Get()
	defer cancel()

	r := ctx.Request
	w := ctx.Writer

	id := r.PostFormValue("id")
	payload := r.PostFormValue("payload")

	count, err := t.mogx.Edit(r.Context(), id, payload)
	if err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		return res.Json(w, http.StatusInternalServerError)

	}
	res.Data = count
	return res.Json(w, http.StatusOK)

}

func (t *EventLog) Retry(ctx *bwebframework.BeanContext) error {

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
