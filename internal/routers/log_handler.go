package routers

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/internal/driver/bredis"
	"github.com/spf13/cast"
	"net/http"
	"strings"
	"time"
)

type Log struct {
	client redis.UniversalClient
	mgo    *bmongo.BMongo
	prefix string
}

func NewLog(client redis.UniversalClient, x *bmongo.BMongo, prefix string) *Log {
	return &Log{client: client, mgo: x, prefix: prefix}
}

// del ,retry,archive,detail
func (t *Log) List(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	id := r.FormValue("id")
	msgType := r.FormValue("msgType")

	if id == "" || msgType == "" {
		// error
		result.Code = berror.MissParameterCode
		result.Msg = berror.MissParameterMsg
		_ = result.Json(w, http.StatusBadRequest)
		return
	}
	data, err := t.detailHandler(r.Context(), id, msgType)
	if err != nil {
		result.Code = "1003"
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	result.Data = data
	_ = result.Json(w, http.StatusOK)
}

func (t *Log) Retry(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	id := r.PostFormValue("id")
	msgType := r.PostFormValue("msgType")
	if msgType == "" {
		msgType = "success"
	}
	if id == "" {
		result.Code = berror.MissParameterCode
		result.Msg = berror.MissParameterMsg
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	if err := t.retryHandler(r.Context(), id, msgType); err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	_ = result.Json(w, http.StatusOK)
}

func (t *Log) Delete(w http.ResponseWriter, r *http.Request) {
	result, cancel := response.Get()
	defer cancel()

	msgType := r.FormValue("msgType")
	score := r.FormValue("score")
	key := strings.Join([]string{t.prefix, "logs", msgType}, ":")

	if err := ZRemRangeByScore(r.Context(), t.client, key, score, score); err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	_ = result.Json(w, http.StatusOK)
}

func (t *Log) Add(w http.ResponseWriter, r *http.Request) {
}

// log detail
func (t *Log) detailHandler(ctx context.Context, id, msgType string) (map[string]any, error) {

	key := strings.Join([]string{t.prefix, "logs", msgType}, ":")

	var build strings.Builder
	build.Grow(3)
	build.WriteString("*")
	build.WriteString(id)
	build.WriteString("*")

	vals, _, err := ZScan(ctx, t.client, key, 0, build.String(), 1)
	if err != nil {
		return nil, err
	}
	if len(vals) <= 0 {
		return nil, errors.New("record is empty")
	}

	m := make(map[string]any)
	if err := json.Unmarshal([]byte(vals[0]), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (t *Log) retryHandler(ctx context.Context, id, msgType string) error {

	key := strings.Join([]string{t.prefix, "logs", msgType}, ":")

	var build strings.Builder
	build.Grow(3)
	build.WriteString("*")
	build.WriteString(id)
	build.WriteString("*")

	keys, _, err := ZScan(ctx, t.client, key, 0, build.String(), 1)
	if err != nil {
		return err
	}
	if len(keys) <= 0 {
		return errors.New("record is empty")
	}

	var data map[string]any

	if err := json.Unmarshal([]byte(keys[0]), &data); err != nil {
		return err
	}

	bk := bredis.NewSchedule(t.client, t.prefix, 10, 20*time.Minute)
	if err := bk.Enqueue(ctx, data); err != nil {
		return err
	}
	return nil
}

func (t *Log) OptLogs(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	query := r.URL.Query()
	page := cast.ToInt64(query.Get("page"))
	pageSize := cast.ToInt64(query.Get("pageSize"))

	data, total, err := t.mgo.OptLogs(r.Context(), page, pageSize)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = map[string]any{"data": data, "total": total, "cursor": page}
	_ = res.Json(w, http.StatusOK)
}

func (t *Log) DelOptLog(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	id := r.URL.Query().Get("id")
	if _, err := t.mgo.DeleteOptLog(r.Context(), id); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	_ = res.Json(w, http.StatusOK)
}

func (t *Log) WorkFlowLogs(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	query := r.URL.Query()
	page := cast.ToInt64(query.Get("page"))
	pageSize := cast.ToInt64(query.Get("pageSize"))

	data, total, err := t.mgo.WorkFlowLogs(r.Context(), nil, page, pageSize)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = map[string]any{"data": data, "total": total, "cursor": page}
	_ = res.Json(w, http.StatusOK)
}
