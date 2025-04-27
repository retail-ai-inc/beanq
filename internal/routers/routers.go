package routers

import (
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/bgzip"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/ui"
)

type Handles struct {
	schedule  *Schedule
	queue     *Queue
	logs      *Logs
	log       *Log
	redisInfo *RedisInfo
	login     *Login
	client    *Client
	dashboard *Dashboard
	eventLog  *EventLog
	user      *User
	dlq       *Dlq
	workflow  *WorkFlow
	role      *Role
	pod       *Pod
}

func NewRouters(mux *http.ServeMux, fs2 fs.FS, modFiles map[string]time.Time, client redis.UniversalClient, mgo *bmongo.BMongo, prefix string, ui ui.Ui) {

	hdls := Handles{
		schedule:  NewSchedule(client, prefix),
		queue:     NewQueue(client, prefix),
		logs:      NewLogs(client, prefix),
		log:       NewLog(client, mgo, prefix),
		redisInfo: NewRedisInfo(client, prefix, mgo),
		login:     NewLogin(client, mgo, prefix, ui),
		client:    NewClient(client, prefix),
		dashboard: NewDashboard(client, mgo, prefix),
		eventLog:  NewEventLog(client, mgo, prefix),
		user:      NewUser(client, mgo, prefix),
		dlq:       NewDlq(client, mgo, prefix),
		workflow:  NewWorkFlow(client, mgo, prefix),
		role:      NewRole(mgo),
		pod:       NewPod(client, mgo, prefix),
	}

	mux.HandleFunc("GET /", func(writer http.ResponseWriter, request *http.Request) {

		fd, err := fs.Sub(fs2, "ui")
		if err != nil {
			log.Fatalf("static files error:%+v \n", err)
		}

		path := request.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		_, err = fs.Stat(fd, strings.TrimLeft(path, "/"))
		if err != nil {
			http.Error(writer, "Not Found", http.StatusNotFound)
			return
		}

		ifModifiedSince := request.Header.Get("If-Modified-Since")
		if ifModifiedSince != "" {
			ifModifiedSinceTime, err := time.ParseInLocation(time.RFC1123, ifModifiedSince, time.UTC)
			if err == nil && modFiles[path].UTC().Before(ifModifiedSinceTime.Add(1*time.Second)) {
				writer.WriteHeader(http.StatusNotModified)
				return
			}
		}
		writer.Header().Set("Last-Modified", modFiles[path].UTC().Format(time.RFC1123))

		handle := http.FileServer(http.FS(fd))

		if !bgzip.MatchGzipEncoding(request) || !strings.Contains(request.URL.Path, ".js") && !strings.Contains(request.URL.Path, ".vue") {
			handle.ServeHTTP(writer, request)
			return
		}

		writer.Header().Set("Content-Encoding", "gzip")
		writer.Header().Set("Vary", "Accept-Encoding")

		gz, err := bgzip.NewGzipResponseWriter(writer)
		if err != nil {
			http.Error(writer, "Not Found", http.StatusNotFound)
			return
		}
		defer gz.Close()

		handle.ServeHTTP(gz, request)
	})

	mux.HandleFunc("GET /ping", HeaderRule(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	}))
	mux.HandleFunc("GET /schedule", MigrateMiddleWare(hdls.schedule.List, client, mgo, prefix, ui))
	mux.HandleFunc("GET /queue/list", MigrateMiddleWare(hdls.queue.List, client, mgo, prefix, ui))
	mux.HandleFunc("GET /queue/detail", MigrateMiddleWare(hdls.queue.Detail, client, mgo, prefix, ui))

	mux.HandleFunc("GET /logs", MigrateMiddleWare(hdls.logs.List, client, mgo, prefix, ui))
	mux.HandleFunc("GET /log", MigrateMiddleWare(hdls.log.List, client, mgo, prefix, ui))
	mux.HandleFunc("GET /log/opt_log", MigrateMiddleWare(hdls.log.OptLogs, client, mgo, prefix, ui))
	mux.HandleFunc("DELETE /log/opt_log", MigrateMiddleWare(hdls.log.DelOptLog, client, mgo, prefix, ui))
	mux.HandleFunc("GET /log/workflow_log", MigrateMiddleWare(hdls.log.WorkFlowLogs, client, mgo, prefix, ui))

	mux.HandleFunc("GET /redis", MigrateSSE(hdls.redisInfo.Info, client, mgo, prefix, ui, "redis_info"))
	mux.HandleFunc("GET /redis/monitor", MigrateSSE(hdls.redisInfo.Monitor, client, mgo, prefix, ui, "redis_monitor"))
	mux.HandleFunc("GET /redis/keys", MigrateMiddleWare(hdls.redisInfo.Keys, client, mgo, prefix, ui))
	mux.HandleFunc("DELETE /redis/{key}", MigrateMiddleWare(hdls.redisInfo.DeleteKey, client, mgo, prefix, ui))
	mux.HandleFunc("PUT /redis/config", MigrateMiddleWare(hdls.redisInfo.Config, client, mgo, prefix, ui))
	mux.HandleFunc("GET /redis/config", MigrateMiddleWare(hdls.redisInfo.ConfigInfo, client, mgo, prefix, ui))

	mux.HandleFunc("POST /test/notify", MigrateMiddleWare(hdls.login.TestNotify, client, mgo, prefix, ui))

	mux.HandleFunc("POST /login", HeaderRule(hdls.login.Login))
	mux.HandleFunc("GET /login/allowGoogle", HeaderRule(hdls.login.LoginAllowGoogle))
	mux.HandleFunc("GET /clients", MigrateMiddleWare(hdls.client.List, client, mgo, prefix, ui))

	mux.HandleFunc("GET /dashboard/graphic", MigrateSSE(hdls.dashboard.Info, client, mgo, prefix, ui, "dashboard"))
	mux.HandleFunc("GET /dashboard/total", MigrateMiddleWare(hdls.dashboard.Total, client, mgo, prefix, ui))
	mux.HandleFunc("GET /dashboard/pods", MigrateMiddleWare(hdls.dashboard.Pods, client, mgo, prefix, ui))
	mux.HandleFunc("GET /nodes", MigrateMiddleWare(hdls.dashboard.Nodes, client, mgo, prefix, ui))

	mux.HandleFunc("GET /event_log/list", MigrateSSE(hdls.eventLog.List, client, mgo, prefix, ui, "event_log"))
	mux.HandleFunc("GET /event_log/detail", MigrateMiddleWare(hdls.eventLog.Detail, client, mgo, prefix, ui))
	mux.HandleFunc("POST /event_log/delete", MigrateMiddleWare(hdls.eventLog.Delete, client, mgo, prefix, ui))
	mux.HandleFunc("POST /event_log/edit", MigrateMiddleWare(hdls.eventLog.Edit, client, mgo, prefix, ui))
	mux.HandleFunc("POST /event_log/retry", MigrateMiddleWare(hdls.eventLog.Retry, client, mgo, prefix, ui))

	mux.HandleFunc("GET /user/list", MigrateMiddleWare(hdls.user.List, client, mgo, prefix, ui))
	mux.HandleFunc("POST /user/add", MigrateMiddleWare(hdls.user.Add, client, mgo, prefix, ui))
	mux.HandleFunc("POST /user/del", MigrateMiddleWare(hdls.user.Delete, client, mgo, prefix, ui))
	mux.HandleFunc("POST /user/edit", MigrateMiddleWare(hdls.user.Edit, client, mgo, prefix, ui))

	mux.HandleFunc("GET /role/list", MigrateMiddleWare(hdls.role.List, nil, mgo, "", ui))
	mux.HandleFunc("POST /role/add", MigrateMiddleWare(hdls.role.Add, nil, mgo, "", ui))
	mux.HandleFunc("POST /role/delete", MigrateMiddleWare(hdls.role.Delete, nil, mgo, "", ui))
	mux.HandleFunc("POST /role/edit", MigrateMiddleWare(hdls.role.Edit, nil, mgo, "", ui))

	mux.HandleFunc("GET /googleLogin", hdls.login.GoogleLogin)
	mux.HandleFunc("GET /callback", hdls.login.GoogleCallBack)

	mux.HandleFunc("GET /dlq/list", MigrateMiddleWare(hdls.dlq.List, client, mgo, prefix, ui))
	mux.HandleFunc("POST /dlq/retry", MigrateMiddleWare(hdls.dlq.Retry, client, mgo, prefix, ui))
	mux.HandleFunc("POST /dlq/delete", MigrateMiddleWare(hdls.dlq.Delete, client, mgo, prefix, ui))
	mux.HandleFunc("GET /workflow/list", MigrateMiddleWare(hdls.workflow.List, client, mgo, prefix, ui))
	mux.HandleFunc("POST /workflow/delete", MigrateMiddleWare(hdls.workflow.Delete, client, mgo, prefix, ui))

	mux.HandleFunc("GET /pod/list", MigrateMiddleWare(hdls.pod.List, client, mgo, prefix, ui))
}
