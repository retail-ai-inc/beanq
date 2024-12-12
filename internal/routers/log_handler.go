package routers

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
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
	prefix string
}

func NewLog(client redis.UniversalClient, prefix string) *Log {
	return &Log{client: client, prefix: prefix}
}

// del ,retry,archive,detail
func (t *Log) List(beanContext *bwebframework.BeanContext) error {

	result, cancel := response.Get()
	defer cancel()
	r := beanContext.Request
	w := beanContext.Writer

	id := r.FormValue("id")
	msgType := r.FormValue("msgType")

	if id == "" || msgType == "" {
		// error
		result.Code = berror.MissParameterCode
		result.Msg = berror.MissParameterMsg
		return result.Json(w, http.StatusBadRequest)
	}
	data, err := t.detailHandler(r.Context(), id, msgType)
	if err != nil {
		result.Code = "1003"
		result.Msg = err.Error()
		return result.Json(w, http.StatusInternalServerError)
	}
	result.Data = data
	return result.Json(w, http.StatusOK)
}

func (t *Log) Retry(beanContext *bwebframework.BeanContext) error {

	result, cancel := response.Get()
	defer cancel()

	r := beanContext.Request
	w := beanContext.Writer

	id := r.PostFormValue("id")
	msgType := r.PostFormValue("msgType")
	if msgType == "" {
		msgType = "success"
	}
	if id == "" {
		result.Code = berror.MissParameterCode
		result.Msg = berror.MissParameterMsg
		return result.Json(w, http.StatusInternalServerError)
	}
	if err := t.retryHandler(r.Context(), id, msgType); err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusInternalServerError)
	}
	return result.Json(w, http.StatusOK)
}

func (t *Log) Delete(beanContext *bwebframework.BeanContext) error {
	result, cancel := response.Get()
	defer cancel()

	w := beanContext.Writer
	r := beanContext.Request

	msgType := r.FormValue("msgType")
	score := r.FormValue("score")
	key := strings.Join([]string{t.prefix, "logs", msgType}, ":")

	if err := ZRemRangeByScore(r.Context(), t.client, key, score, score); err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusInternalServerError)
	}
	return result.Json(w, http.StatusOK)
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

func (t *Log) OptLogs(beanContext *bwebframework.BeanContext) error {

	res, cancel := response.Get()
	defer cancel()

	w := beanContext.Writer
	r := beanContext.Request

	key := strings.Join([]string{t.prefix, "beanq-logic-log"}, ":")
	result := t.client.XRangeN(r.Context(), key, "-", "+", 10).Val()
	data := make([]map[string]any, 0, len(result))

	for _, value := range result {
		logType, ok := value.Values["logType"]
		if !ok {
			continue
		}
		if cast.ToString(logType) != "opt" {
			continue
		}
		data = append(data, value.Values)

	}
	res.Data = data
	return res.Json(w, http.StatusOK)
}
