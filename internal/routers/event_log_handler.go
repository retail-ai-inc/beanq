package routers

import (
	"net/http"
	"sort"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	public "github.com/retail-ai-inc/beanq/v3/internal"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/retail-ai-inc/beanq/v3/internal/driver/bredis"
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
	query := r.URL.Query()
	page := cast.ToInt64(query.Get("page"))
	pageSize := cast.ToInt64(query.Get("pageSize"))
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
	if page <= 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	flush, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "server err", http.StatusInternalServerError)
		return
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
			return
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
	moodType := ""
	if v, ok := data["moodType"]; ok {
		moodType = v.(string)
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

	var bk public.IBroker
	if moodType == string(btype.SEQUENTIAL) {
		_ = res.Json(w, http.StatusOK)
		return
	}
	if moodType == string(btype.DELAY) {

		bk = bredis.NewSchedule(t.client, t.prefix, 10, 20*time.Minute)
		if err := bk.Enqueue(nctx, data); err != nil {
			res.Msg = err.Error()
			res.Code = berror.InternalServerErrorCode
			_ = res.Json(w, http.StatusOK)
			return
		}
		_ = res.Json(w, http.StatusOK)
		return
	}

	bk = bredis.NewNormal(t.client, t.prefix, 2000, 10, 20*time.Minute)
	if err := bk.Enqueue(nctx, data); err != nil {
		res.Msg = err.Error()
		res.Code = berror.InternalServerErrorCode
		_ = res.Json(w, http.StatusOK)
		return
	}
	_ = res.Json(w, http.StatusOK)
}
