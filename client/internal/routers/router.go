package routers

import (
	"log"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"beanq/client/helper/jsonx"
	"beanq/client/internal/redisx"
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

	var data map[string]any
	data = make(map[string]any, 3)
	data["errorCode"] = "0000"
	data["errorMsg"] = "success"

	client := redisx.RClient("127.0.0.1:6381", "secret", 0)
	defer client.Close()

	ctx := request.Context()
	// get queues
	queues, err := redisx.Keys(ctx, client, "beanq:*:zset")
	if err != nil {
		log.Fatal(err)
		return
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
		d = append(d, map[string]any{"queue": queue, "state": "Run", "size": objStr.SerizlizedLength, "memory": r, "process": 112067, "fail": 0, "errRate": "2%"})
	}

	data["data"] = d
	bt, _ := jsonx.Marshal(&data)
	writer.Write(bt)
	return
}
func QueueHandler(writer http.ResponseWriter, request *http.Request) {

	var data map[string]any
	data = make(map[string]any, 3)
	data["errorCode"] = "0000"
	data["errorMsg"] = "success"

	client := redisx.RClient("127.0.0.1:6381", "secret", 0)
	defer client.Close()

	ctx := request.Context()
	// get queues
	queues, err := redisx.Keys(ctx, client, "beanq:*:stream")
	if err != nil {
		log.Fatal(err)
		return
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
	bt, _ := jsonx.Marshal(&data)
	writer.Write(bt)
	return
}
func RedisHandler(w http.ResponseWriter, r *http.Request) {
	var data map[string]any
	data = make(map[string]any, 3)
	data["errorCode"] = "0000"
	data["errorMsg"] = "success"

	client := redisx.RClient("127.0.0.1:6381", "secret", 0)
	defer client.Close()
	d, err := redisx.Info(r.Context(), client)
	if err != nil {
		log.Println(err)
		return
	}
	data["data"] = d
	bt, _ := jsonx.Marshal(&data)
	w.Write(bt)
	return
}
