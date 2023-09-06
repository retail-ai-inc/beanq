package routers

import (
	"context"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/client/internal/jwtx"
	"github.com/retail-ai-inc/beanq/client/internal/redisx"
	"github.com/retail-ai-inc/beanq/client/internal/routers/consts"
	"github.com/retail-ai-inc/beanq/client/internal/simple_router"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/retail-ai-inc/beanq/internal/options"
	"github.com/spf13/cast"
)

func IndexHandler(ctx *simple_router.Context) error {

	url := ctx.Request().RequestURI
	if strings.HasSuffix(url, ".vue") {
		ctx.Response().Header().Set("Content-Type", "application/octet-stream")
	}
	var dir string = "./"
	_, f, _, ok := runtime.Caller(0)
	if ok {
		dir = filepath.Dir(f)
	}

	hdl := http.FileServer(http.Dir(path.Join(dir, "../../ui/")))
	hdl.ServeHTTP(ctx.Response(), ctx.Request())
	return nil
}

func LoginHandler(ctx *simple_router.Context) error {

	// request := ctx.Request()
	// username := request.PostFormValue("username")
	// password := request.PostFormValue("password")

	result := resultPool.Get().(*Result)
	defer func() {
		result.Reset()
		resultPool.Put(result)
	}()
	claim := jwt.RegisteredClaims{
		Issuer:    "",
		Subject:   "beanq monitor ui",
		Audience:  nil,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7200 * time.Second)),
		NotBefore: nil,
		IssuedAt:  nil,
		ID:        "",
	}
	token, err := jwtx.MakeRsaToken(claim)
	if err != nil {

	}

	result.Data = map[string]any{"token": token}
	return ctx.Json(http.StatusOK, result)

}

func ScheduleHandler(ctx *simple_router.Context) error {

	result := resultPool.Get().(*Result)
	defer func() {
		result.Reset()
		resultPool.Put(result)
	}()

	bt, err := queueInfo(ctx.Context(), redisx.ScheduleQueueKey("beanq"))

	if err != nil {
		result.Code = consts.InternalServerErrorCode
		result.Msg = err.Error()
		return ctx.Json(http.StatusInternalServerError, result)
	}
	result.Data = bt
	return ctx.Json(http.StatusOK, result)
}

func QueueHandler(ctx *simple_router.Context) error {

	result := resultPool.Get().(*Result)
	defer func() {
		result.Reset()
		resultPool.Put(result)
	}()
	nctx := ctx.Context()
	bt, err := queueInfo(nctx, redisx.QueueKey("beanq"))
	if err != nil {
		result.Code = consts.InternalServerErrorCode
		result.Msg = err.Error()
		return ctx.Json(http.StatusInternalServerError, result)
	}

	result.Data = bt

	return ctx.Json(http.StatusOK, result)
}

func LogArchiveHandler(ctx *simple_router.Context) error {
	result := resultPool.Get().(*Result)
	defer func() {
		result.Reset()
		resultPool.Put(result)
	}()

	return ctx.Json(http.StatusOK, result)
}

func LogRetryHandler(ctx *simple_router.Context) error {
	result := resultPool.Get().(*Result)
	defer func() {
		result.Reset()
		resultPool.Put(result)
	}()
	req := ctx.Request()
	id := req.PostFormValue("id")
	if id == "" {
		result.Code = consts.MissParameterCode
		result.Msg = consts.MissParameterMsg
		return ctx.Json(http.StatusInternalServerError, result)
	}
	client := redisx.Client(redisx.Addr, redisx.PassWord, redisx.Db)

	nid := cast.ToInt64(id)

	cmd := client.ZRange(ctx.Context(), "beanq:logs:success", nid, nid)
	if err := cmd.Err(); err != nil {
		result.Code = consts.InternalServerErrorCode
		result.Msg = err.Error()
		return ctx.Json(http.StatusInternalServerError, result)
	}
	vals := cmd.Val()
	if len(vals) < 1 {
		result.Code = consts.InternalServerErrorCode
		result.Msg = "record is empty"
		return ctx.Json(http.StatusInternalServerError, result)
	}
	valByte := []byte(vals[0])

	newJson := json.Json
	payload := newJson.Get(valByte, "Payload").ToString()
	executeTime := newJson.Get(valByte, "ExecuteTime").ToString()
	groupName := newJson.Get(valByte, "Group").ToString()
	queue := newJson.Get(valByte, "Queue").ToString()
	queues := strings.Split(queue, ":")
	if len(queues) < 4 {
		result.Code = consts.InternalServerErrorCode
		result.Msg = "data error"
		return ctx.Json(http.StatusInternalServerError, result)
	}

	dup, err := time.ParseInLocation(time.RFC3339, executeTime, time.Local)
	if err != nil {
		result.Code = consts.InternalServerErrorCode
		result.Msg = err.Error()
		return ctx.Json(http.StatusInternalServerError, result)
	}
	publish := beanq.NewPublisher()
	task := beanq.NewTask([]byte(payload))
	if err := publish.PublishWithContext(ctx.Context(), task, options.ExecuteTime(dup), options.Group(groupName), options.Queue(queues[2])); err != nil {
		result.Code = consts.InternalServerErrorCode
		result.Msg = err.Error()
		return ctx.Json(http.StatusInternalServerError, result)
	}

	return ctx.Json(http.StatusOK, result)
}

func LogDelHandler(ctx *simple_router.Context) error {

	result := resultPool.Get().(*Result)
	defer func() {
		result.Reset()
		resultPool.Put(result)
	}()
	req := ctx.Request()
	id := req.FormValue("id")
	if id == "" {
		result.Code = consts.MissParameterCode
		result.Msg = consts.MissParameterMsg
		return ctx.Json(http.StatusInternalServerError, result)
	}

	client := redisx.Client(redisx.Addr, redisx.PassWord, redisx.Db)

	nid := cast.ToInt64(id)
	var start int64
	start = nid - 1
	if start <= 0 {
		start = 0
	}

	cmd := client.ZRemRangeByRank(ctx.Context(), "beanq:logs:success", start, nid)

	if cmd.Err() != nil {
		result.Code = consts.InternalServerErrorCode
		result.Msg = cmd.Err().Error()
		return ctx.Json(http.StatusInternalServerError, result)
	}

	return ctx.Json(http.StatusOK, result)
}

func LogHandler(ctx *simple_router.Context) error {

	resultRes := resultPool.Get().(*Result)
	defer func() {
		resultRes.Reset()
		resultPool.Put(resultRes)
	}()

	client := redisx.Client(redisx.Addr, redisx.PassWord, redisx.Db)

	var (
		page, pageSize int64
		dataType       string = "success"
		matchStr       string = "beanq:logs:success"
		// replaeceStr    string = "beanq:logs:success:"
	)

	req := ctx.Request()
	page = cast.ToInt64(req.FormValue("page"))
	pageSize = cast.ToInt64(req.FormValue("pageSize"))
	dataType = req.FormValue("type")

	if dataType != "success" && dataType != "error" {
		resultRes.Code = consts.TypeErrorCode
		resultRes.Msg = consts.TypeErrorMsg

		return ctx.Json(http.StatusInternalServerError, resultRes)

	}

	nowPage := (page - 1) * pageSize
	if nowPage <= 0 {
		nowPage = 0
	}
	nowPageSize := page * pageSize
	if nowPageSize <= 0 {
		nowPageSize = 9
	}

	if dataType == "error" {
		matchStr = "beanq:logs:error"
		// replaeceStr = "beanq:logs:error:"
	}
	nctx := ctx.Context()
	cmd := client.ZRange(nctx, matchStr, nowPage, nowPageSize)
	if cmd.Err() != nil {
		resultRes.Msg = cmd.Err().Error()
		resultRes.Code = consts.InternalServerErrorCode
		return ctx.Json(http.StatusInternalServerError, resultRes)
	}

	result, err := cmd.Result()
	if err != nil {
		resultRes.Msg = cmd.Err().Error()
		resultRes.Code = consts.InternalServerErrorCode
		return ctx.Json(http.StatusInternalServerError, resultRes)
	}

	njson := json.Json

	length, err := client.ZLexCount(nctx, matchStr, "-", "+").Result()
	if err != nil {
		resultRes.Msg = err.Error()
		resultRes.Code = consts.InternalServerErrorCode
		return ctx.Json(http.StatusInternalServerError, resultRes)
	}
	d := make([]map[string]any, 0, pageSize)
	for _, v := range result {

		cmd := client.ZRank(nctx, matchStr, v)
		key, err := cmd.Result()
		if err != nil {
			continue
		}
		payloadByte := stringx.StringToByte(v)
		npayload := njson.Get(payloadByte, "Payload")
		addTime := njson.Get(payloadByte, "AddTime")
		runTime := njson.Get(payloadByte, "RunTime")
		group := njson.Get(payloadByte, "Group")
		queue := njson.Get(payloadByte, "Queue")

		ttl := cast.ToTime(njson.Get(payloadByte, "ExpireTime").ToString()).Sub(time.Now()).Seconds()
		d = append(d, map[string]any{"key": key, "ttl": ttl, "addTime": addTime, "runTime": runTime, "group": group, "queue": queue, "payload": npayload})

	}
	resultRes.Data = map[string]any{"data": d, "total": length}

	return ctx.Json(http.StatusOK, resultRes)
}

func RedisHandler(ctx *simple_router.Context) error {
	result := resultPool.Get().(*Result)

	defer func() {
		result.Reset()
		resultPool.Put(result)
	}()

	client := redisx.Client(redisx.Addr, redisx.PassWord, redisx.Db)

	d, err := redisx.Info(ctx.Context(), client)
	if err != nil {
		result.Code = "1001"
		result.Msg = err.Error()
		return ctx.Json(http.StatusInternalServerError, result)
	}

	result.Data = d

	return ctx.Json(http.StatusOK, result)
}

func queueInfo(ctx context.Context, queueKey string) (any, error) {

	client := redisx.Client(redisx.Addr, redisx.PassWord, redisx.Db)

	// get queues
	queues, err := redisx.Keys(ctx, client, queueKey)
	if err != nil {
		return nil, err
	}
	d := make([]map[string]any, 0, len(queues))
	for _, queue := range queues {
		objStr := redisx.Object(ctx, client, queue)
		// get memory
		r, err := client.MemoryUsage(ctx, queue).Result()
		if err != nil {
			log.Println(err)
			continue
		}
		d = append(d, map[string]any{"queue": queue, "state": "Run", "size": objStr.SerizlizedLength, "memory": r, "process": objStr.LruSecondsIdle, "fail": 0, "errRate": "2%"})
	}

	return d, nil
}
