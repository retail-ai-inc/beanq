package routers

import (
	"net/http"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
)

type EventLog struct {
	Id     string `json:"id"`
	client redis.UniversalClient
	mogx   *bmongo.BMongo
	prefix string
}

func NewEventLog(client redis.UniversalClient, x *bmongo.BMongo, prefix string) *EventLog {
	return &EventLog{client: client, mogx: x, prefix: prefix}
}

func (t *EventLog) List(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer func() {
		cancel()
	}()

	eventName := cast.ToString(r.Context().Value(EventName{}))

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
	filter["logType"] = bstatus.Logic
	if id != "" {
		filter["id"] = id
	}
	if moodType != "" {
		filter["moodType"] = moodType
	}
	if topicName != "" {
		filter["topic"] = topicName
	}
	if status != "" {
		statusValid := []string{"failed", "published", "success"}
		if index := sort.SearchStrings(statusValid, status); index < len(statusValid) && statusValid[index] == status {
			filter["status"] = status
		} else {
			http.Error(w, "Invalid status value", http.StatusBadRequest)
			return
		}
	}
	datas := make(map[string]any, 3)
	nctx := r.Context()
	emit := func() error {
		data, total, err := t.mogx.EventLogs(nctx, filter, page.Page, page.PageSize)
		if err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
		} else {
			result.Code = berror.SuccessCode
			result.Msg = ""
			datas["data"] = data
			datas["total"] = total
			datas["cursor"] = page.Page
			result.Data = datas
		}
		return result.EventMsg(w, eventName)
	}
	if err := streamEvents(w, r, 5*time.Second, emit); err != nil {
		return
	}
}

func (t *EventLog) Detail(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	id := r.URL.Query().Get("id")
	data, err := t.mogx.DetailEventLog(r.Context(), id)
	if err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		_ = res.Json(w, http.StatusInternalServerError)
		return

	}
	res.Data = data
	_ = res.Json(w, http.StatusOK)
}

func (t *EventLog) Delete(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	id := r.PostFormValue("id")
	count, err := t.mogx.Delete(r.Context(), id)
	if err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	if count == 0 {
		writeAPIError(w, http.StatusNotFound, berror.MissParameterCode, "event not found")
		return
	}
	res.Data = count
	_ = res.Json(w, http.StatusOK)
}

func (t *EventLog) Edit(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	id := r.PostFormValue("id")
	payload := r.PostFormValue("payload")

	count, err := t.mogx.Edit(r.Context(), id, payload)
	if err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		_ = res.Json(w, http.StatusInternalServerError)
		return

	}
	if count == 0 {
		writeAPIError(w, http.StatusNotFound, berror.MissParameterCode, "event not found")
		return
	}
	res.Data = count
	_ = res.Json(w, http.StatusOK)
}

func (t *EventLog) Retry(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	m := make(map[string]any)
	id := r.FormValue("id")
	m["uniqueId"] = id
	nctx := r.Context()

	data := make(map[string]any)
	if err := json.Unmarshal([]byte(r.FormValue("data")), &data); err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		_ = res.Json(w, http.StatusInternalServerError)
		return

	}
	// only failed messages can be retried
	if v, ok := data["status"]; ok {
		if cast.ToString(v) != bstatus.StatusFailed {
			res.Msg = "Only failed messages can be retried"
			res.Code = berror.SuccessCode
			_ = res.Json(w, http.StatusOK)
			return
		}
	}
	if _, ok := data["addTime"]; ok {
		data["addTime"] = time.Now()
	}
	delete(data, "beginTime")
	delete(data, "endTime")
	if _, ok := data["retry"]; ok {
		data["retry"] = 0
	}
	delete(data, "runTime")
	uniqueId := ""
	if v, ok := data["id"]; ok {
		uniqueId = cast.ToString(v)
	}

	b, err := t.mogx.EventRetryCheck(nctx, uniqueId)
	if err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	if !b {
		res.Msg = berror.PreventMultipleRetryMsg
		res.Code = berror.PreventMultipleRetryCode
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}

	if err := publishRetry(nctx, data, defaultRetryPublisherFactory(t.client, t.prefix)); err != nil {
		res.Msg = err.Error()
		res.Code = berror.TypeErrorCode
		_ = res.Json(w, http.StatusBadRequest)
		return
	}
	_ = res.Json(w, http.StatusOK)
}
