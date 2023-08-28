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
	"github.com/retail-ai-inc/beanq/client/internal/jwtx"
	"github.com/retail-ai-inc/beanq/client/internal/redisx"
	"github.com/retail-ai-inc/beanq/client/internal/simple_router"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/stringx"
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
		result.Code = "1001"
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
		result.Code = "1001"
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
		result.Code = "1000"
		result.Msg = "missing parameter"
		return ctx.Json(http.StatusInternalServerError, result)
	}
	// client := redisx.Client(redisx.Addr, redisx.PassWord, redisx.Db)

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
		result.Code = "1000"
		result.Msg = "missing parameter"
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
		result.Code = "1000"
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
		resultRes.Code = "1001"
		resultRes.Msg = "type is error"

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
		resultRes.Code = "1001"
		return ctx.Json(http.StatusInternalServerError, resultRes)
	}

	result, err := cmd.Result()
	if err != nil {
		resultRes.Msg = cmd.Err().Error()
		resultRes.Code = "1001"
		return ctx.Json(http.StatusInternalServerError, resultRes)
	}

	json := json.Json

	len, err := client.ZLexCount(nctx, matchStr, "-", "+").Result()
	if err != nil {
		resultRes.Msg = err.Error()
		resultRes.Code = "1001"
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
		npayload := json.Get(payloadByte, "Payload")
		addTime := json.Get(payloadByte, "AddTime")
		runTime := json.Get(payloadByte, "RunTime")
		group := json.Get(payloadByte, "Group")
		queue := json.Get(payloadByte, "Queue")

		ttl := cast.ToTime(json.Get(payloadByte, "ExpireTime").ToString()).Sub(time.Now()).Seconds()
		d = append(d, map[string]any{"key": key, "ttl": ttl, "addTime": addTime, "runTime": runTime, "group": group, "queue": queue, "payload": npayload})

	}
	resultRes.Data = map[string]any{"data": d, "total": len}

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
