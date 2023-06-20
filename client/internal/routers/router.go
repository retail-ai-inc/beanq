package routers

import (
	"context"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"beanq/client/helper/jsonx"
	"beanq/client/internal/redisx"
	"beanq/helper/stringx"
	"github.com/spf13/cast"
)

const (
	ScheduleQueueKey = "beanq:*:zset"
	QueueKey         = "beanq:*:stream"
)

func IndexHandler(writer http.ResponseWriter, request *http.Request) {
	url := request.RequestURI
	if strings.HasSuffix(url, ".vue") {
		writer.Header().Set("Content-Type", "application/octet-stream")
	}
	var dir string = "./"
	_, f, _, ok := runtime.Caller(0)
	if ok {
		dir = filepath.Dir(f)
	}

	hdl := http.FileServer(http.Dir(path.Join(dir, "../../ui/")))
	hdl.ServeHTTP(writer, request)
	return
}

func ScheduleHandler(writer http.ResponseWriter, request *http.Request) {

	bt, err := queueInfo(request.Context(), ScheduleQueueKey)
	if err != nil {
		log.Println(err)
		return
	}
	writer.Write(bt)
	return
}
func QueueHandler(writer http.ResponseWriter, request *http.Request) {
	bt, err := queueInfo(request.Context(), QueueKey)
	if err != nil {
		log.Println(err)
		return
	}
	writer.Write(bt)
	return
}
func LogHandler(w http.ResponseWriter, r *http.Request) {

	client := redisx.RClient("127.0.0.1:6381", "secret", 0)
	defer client.Close()

	ctx := r.Context()

	var (
		page, pageSize uint64
		dataType       string = "success"
		matchStr       string = "beanq:logs:success:*"
		replaeceStr    string = "beanq:logs:success:"
	)
	page = cast.ToUint64(r.FormValue("page"))
	pageSize = cast.ToUint64(r.FormValue("pageSize"))
	dataType = r.FormValue("type")

	if pageSize <= 0 {
		pageSize = 10
	}
	if dataType == "error" {
		matchStr = "beanq:logs:error:*"
		replaeceStr = "beanq:logs:error:"
	}

	cmd := client.Scan(ctx, page, matchStr, cast.ToInt64(pageSize))
	if cmd.Err() != nil {
		return
	}
	keys, _, err := cmd.Result()
	if err != nil {
		return
	}
	data := make(map[string]any, 3)
	data["errorCode"] = "0000"
	data["errorMsg"] = "success"
	json := jsonx.Json

	d := make([]map[string]any, 0, pageSize)
	for _, key := range keys {
		ttl, _ := client.TTL(ctx, key).Result()
		payload, _ := client.Get(ctx, key).Result()
		nkey := strings.ReplaceAll(key, replaeceStr, "")

		payloadByte := stringx.StringToByte(payload)
		npayload := json.Get(payloadByte, "Payload")
		addTime := json.Get(payloadByte, "AddTime")
		runTime := json.Get(payloadByte, "RunTime")
		group := json.Get(payloadByte, "Group")
		queue := json.Get(payloadByte, "Queue")

		d = append(d, map[string]any{"key": nkey, "ttl": ttl.Seconds(), "addTime": addTime, "runTime": runTime, "group": group, "queue": queue, "payload": npayload})

	}
	data["data"] = d
	bt, _ := jsonx.Marshal(&data)
	w.Write(bt)
	return
}

func RedisHandler(writer http.ResponseWriter, request *http.Request) {
	var data map[string]any
	data = make(map[string]any, 3)
	data["errorCode"] = "0000"
	data["errorMsg"] = "success"

	client := redisx.RClient("127.0.0.1:6381", "secret", 0)
	defer client.Close()
	d, err := redisx.Info(request.Context(), client)
	if err != nil {
		log.Println(err)
		return
	}
	data["data"] = d
	bt, _ := jsonx.Marshal(&data)
	writer.Write(bt)
	return
}
func queueInfo(ctx context.Context, queueKey string) ([]byte, error) {

	data := make(map[string]any, 3)
	data["errorCode"] = "0000"
	data["errorMsg"] = "success"

	client := redisx.RClient("127.0.0.1:6381", "secret", 0)
	defer client.Close()

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

	data["data"] = d
	bt, err := jsonx.Marshal(&data)
	if err != nil {
		return nil, err
	}
	return bt, nil
}
