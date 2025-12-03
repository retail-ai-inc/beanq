package routers

import (
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/bgzip"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/ui"
	"go.mongodb.org/mongo-driver/mongo"
)

type EventName struct {
}
type Handles struct {
	schedule     *Schedule
	queue        *Queue
	logs         *Logs
	log          *Log
	redisInfo    *RedisInfo
	login        *Login
	client       *Client
	dashboard    *Dashboard
	eventLog     *EventLog
	user         *User
	dlq          *Dlq
	workflow     *WorkFlow
	role         *Role
	pod          *Pod
	sequenceLock *SequenceLock
}

type Router struct {
	Mux *http.ServeMux
}
type HandleFunc func(w http.ResponseWriter, r *http.Request)
type Middleware func(HandleFunc) HandleFunc

func NewRouter() *Router {
	return &Router{http.NewServeMux()}
}
func (r *Router) HandleFunc(pattern string, handler HandleFunc, middles ...Middleware) {
	for _, middle := range middles {
		handler = middle(handler)
	}
	r.Mux.HandleFunc(pattern, handler)
}
func RouterList(fs2 fs.FS,
	modFiles map[string]time.Time,
	client redis.UniversalClient,
	mgo *bmongo.BMongo,
	workflowCollection *mongo.Collection,
	prefix string, ui ui.Ui) *Router {

	hdls := Handles{
		schedule:     NewSchedule(client, prefix),
		queue:        NewQueue(client, prefix),
		logs:         NewLogs(client, prefix),
		log:          NewLog(client, mgo, prefix),
		redisInfo:    NewRedisInfo(client, prefix, mgo),
		login:        NewLogin(client, mgo, prefix, ui),
		client:       NewClient(client, prefix),
		dashboard:    NewDashboard(client, mgo, prefix),
		eventLog:     NewEventLog(client, mgo, prefix),
		user:         NewUser(client, mgo, prefix, ui),
		dlq:          NewDlq(client, mgo, prefix),
		workflow:     NewWorkFlow(workflowCollection),
		role:         NewRole(mgo),
		pod:          NewPod(client, mgo, prefix),
		sequenceLock: NewSequenceLock(client, prefix),
	}

	router := NewRouter()
	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fd, err := fs.Sub(fs2, "ui")
		if err != nil {
			log.Fatalf("static files error:%+v \n", err)
		}

		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		_, err = fs.Stat(fd, strings.TrimLeft(path, "/"))
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		ifModifiedSince := r.Header.Get("If-Modified-Since")
		if ifModifiedSince != "" {
			ifModifiedSinceTime, err := time.ParseInLocation(time.RFC1123, ifModifiedSince, time.UTC)
			if err == nil && modFiles[path].UTC().Before(ifModifiedSinceTime.Add(1*time.Second)) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		w.Header().Set("Last-Modified", modFiles[path].UTC().Format(time.RFC1123))

		handle := http.FileServer(http.FS(fd))

		if !bgzip.MatchGzipEncoding(r) || !strings.Contains(r.URL.Path, ".js") && !strings.Contains(r.URL.Path, ".vue") {
			handle.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")

		gz, err := bgzip.NewGzipResponseWriter(w)
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		defer gz.Close()

		handle.ServeHTTP(gz, r)
	})

	router.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	}, HeaderRule())
	router.HandleFunc("GET /schedule", hdls.schedule.List, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("GET /queue/list", hdls.queue.List, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("GET /queue/detail", hdls.queue.Detail, HeaderRule(), AuthSSE(mgo, ui, "queue_detail"))

	router.HandleFunc("GET /logs", hdls.logs.List, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("GET /log", hdls.log.List, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("GET /log/opt_log", hdls.log.OptLogs, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("DELETE /log/opt_log", hdls.log.DelOptLog, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("GET /log/workflow_log", hdls.log.WorkFlowLogs, HeaderRule(), Auth(mgo, ui))

	router.HandleFunc("GET /redis", hdls.redisInfo.Info, HeaderRule(), AuthSSE(mgo, ui, "redis_info"))
	router.HandleFunc("GET /redis/monitor", hdls.redisInfo.Monitor, HeaderRule(), AuthSSE(mgo, ui, "redis_monitor"))
	router.HandleFunc("GET /redis/keys", hdls.redisInfo.Keys, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("DELETE /redis/{key}", hdls.redisInfo.DeleteKey, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("PUT /redis/config", hdls.redisInfo.Config, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("GET /redis/config", hdls.redisInfo.ConfigInfo, HeaderRule(), Auth(mgo, ui))

	router.HandleFunc("POST /test/notify", hdls.login.TestNotify, HeaderRule(), Auth(mgo, ui))

	router.HandleFunc("POST /login", hdls.login.Login, HeaderRule())
	router.HandleFunc("GET /login/allowGoogle", hdls.login.LoginAllowGoogle, HeaderRule())
	router.HandleFunc("GET /clients", hdls.client.List, HeaderRule(), Auth(mgo, ui))

	router.HandleFunc("GET /dashboard/graphic", hdls.dashboard.Info, HeaderRule(), AuthSSE(mgo, ui, "dashboard"))
	router.HandleFunc("GET /dashboard/total", hdls.dashboard.Total, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("GET /dashboard/pods", hdls.dashboard.Pods, HeaderRule(), AuthSSE(mgo, ui, "pods"))
	router.HandleFunc("GET /nodes", hdls.dashboard.Nodes, HeaderRule(), Auth(mgo, ui))

	router.HandleFunc("GET /event_log/list", hdls.eventLog.List, HeaderRule(), AuthSSE(mgo, ui, "event_log"))
	router.HandleFunc("GET /event_log/detail", hdls.eventLog.Detail, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /event_log/delete", hdls.eventLog.Delete, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /event_log/edit", hdls.eventLog.Edit, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /event_log/retry", hdls.eventLog.Retry, HeaderRule(), Auth(mgo, ui))

	router.HandleFunc("GET /sequenceLock/list", hdls.sequenceLock.List, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("DELETE /sequenceLock/unlock/{key}", hdls.sequenceLock.UnLock, HeaderRule(), Auth(mgo, ui))

	router.HandleFunc("GET /user/list", hdls.user.List, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /user/add", hdls.user.Add, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /user/del", hdls.user.Delete, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /user/edit", hdls.user.Edit, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /user/check", hdls.user.Check, HeaderRule(), Auth(mgo, ui))

	router.HandleFunc("GET /role/list", hdls.role.List, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /role/add", hdls.role.Add, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /role/delete", hdls.role.Delete, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /role/edit", hdls.role.Edit, HeaderRule(), Auth(mgo, ui))

	router.HandleFunc("GET /googleLogin", hdls.login.GoogleLogin)
	router.HandleFunc("GET /callback", hdls.login.GoogleCallBack)

	router.HandleFunc("GET /dlq/list", hdls.dlq.List, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /dlq/retry", hdls.dlq.Retry, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /dlq/delete", hdls.dlq.Delete, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("GET /workflow/list", hdls.workflow.List, HeaderRule(), Auth(mgo, ui))
	router.HandleFunc("POST /workflow/delete", hdls.workflow.Delete, HeaderRule(), Auth(mgo, ui))

	router.HandleFunc("GET /pod/list", hdls.pod.List, HeaderRule(), Auth(mgo, ui))

	return router
}
