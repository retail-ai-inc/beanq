package routers

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/bgzip"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/ui"
	"go.mongodb.org/mongo-driver/mongo"
)

func RedisRouterList(fs2 fs.FS, modFiles map[string]time.Time, driver any, mgo *bmongo.BMongo,
	workflowCollection *mongo.Collection, prefix string, uiConfig ui.Ui) (*Router, error) {
	client, ok := driver.(redis.UniversalClient)
	if !ok {
		return nil, fmt.Errorf("ui requires redis driver")
	}
	return RouterList(fs2, modFiles, client, mgo, workflowCollection, prefix, uiConfig), nil
}

type EventName struct {
}
type Handles struct {
	schedule  *Schedule
	queue     *Queue
	logs      *Logs
	log       *Log
	redisInfo *RedisInfo
	mongoInfo *MongoInfo
	login     *Login
	client    *Client
	dashboard *Dashboard
	eventLog  *EventLog
	user      *User
	dlq       *Dlq
	workflow  *WorkFlow
	role      *Role
	pod       *Pod
	tenant    *Tenants
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
	for i := len(middles) - 1; i >= 0; i-- {
		handler = middles[i](handler)
	}
	handler = HeaderRule()(Recover()(handler))
	r.Mux.HandleFunc(pattern, handler)
}
func RouterList(fs2 fs.FS,
	modFiles map[string]time.Time,
	client redis.UniversalClient,
	mgo *bmongo.BMongo,
	workflowCollection *mongo.Collection,
	prefix string, ui ui.Ui) *Router {

	hdls := Handles{
		schedule:  NewSchedule(client, prefix),
		queue:     NewQueue(client, prefix),
		logs:      NewLogs(client, prefix),
		log:       NewLog(client, mgo, prefix),
		redisInfo: NewRedisInfo(client, prefix, mgo),
		mongoInfo: NewMongoInfo(mgo),
		login:     NewLogin(client, mgo, prefix, ui),
		client:    NewClient(client, prefix),
		dashboard: NewDashboard(client, mgo, prefix),
		eventLog:  NewEventLog(client, mgo, prefix),
		user:      NewUser(client, mgo, prefix, ui),
		dlq:       NewDlq(client, mgo, prefix),
		workflow:  NewWorkFlow(workflowCollection),
		role:      NewRole(mgo),
		pod:       NewPod(client, mgo, prefix),
		tenant:    NewTenants(mgo),
	}

	router := NewRouter()
	uiFS, uiFSErr := fs.Sub(fs2, "ui")
	var staticHandler http.Handler
	if uiFSErr == nil {
		staticHandler = http.FileServer(http.FS(uiFS))
	}
	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if uiFSErr != nil {
			http.Error(w, "static files error", http.StatusInternalServerError)
			return
		}

		assetPath := r.URL.Path
		if assetPath == "/" {
			assetPath = "/index.html"
		}
		_, err := fs.Stat(uiFS, strings.TrimLeft(assetPath, "/"))
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		modTime, ok := modFiles[assetPath]
		if !ok {
			http.Error(w, "static file metadata missing", http.StatusInternalServerError)
			return
		}
		lastModified := modTime.UTC().Format(http.TimeFormat)
		etag := fmt.Sprintf("\"%d\"", modTime.UnixNano())
		w.Header().Set("Last-Modified", lastModified)
		w.Header().Set("ETag", etag)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if assetPath == "/index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")
		}

		if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
			if ifNoneMatch == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		} else if ifModifiedSince := r.Header.Get("If-Modified-Since"); ifModifiedSince != "" {
			ifModifiedSinceTime, err := http.ParseTime(ifModifiedSince)
			if err == nil && !modTime.After(ifModifiedSinceTime.Add(time.Second)) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		extension := strings.ToLower(path.Ext(assetPath))
		if !bgzip.MatchGzipEncoding(r) || (extension != ".js" && extension != ".vue") {
			staticHandler.ServeHTTP(w, r)
			return
		}

		gz, err := bgzip.NewGzipResponseWriter(w)
		if err != nil {
			http.Error(w, "gzip initialization error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		defer gz.Close()

		staticHandler.ServeHTTP(gz, r)
	})

	router.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	}, HeaderRule())
	router.HandleFunc("POST /api/v1/auth/login", hdls.login.Login)
	router.HandleFunc("POST /api/v1/auth/logout", hdls.login.Logout)
	router.HandleFunc("GET /api/v1/auth/google", hdls.login.GoogleLogin)
	router.HandleFunc("GET /api/v1/auth/google/callback", hdls.login.GoogleCallBack)
	router.HandleFunc("GET /api/v1/auth/google/config", hdls.login.LoginAllowGoogle, requireMongo(mgo))
	// Compatibility endpoint used by UI clients released before the /api/v1 migration.
	router.HandleFunc("GET /login/allowGoogle", hdls.login.LoginAllowGoogle, requireMongo(mgo))

	auth := Auth(mgo, ui)
	router.HandleFunc("GET /api/v1/schedules", hdls.schedule.List, auth)
	router.HandleFunc("GET /api/v1/queues", hdls.queue.List, auth)
	router.HandleFunc("GET /api/v1/queues/{id}/stream", hdls.queue.Detail, AuthSSE(mgo, ui, "queue_detail"))
	router.HandleFunc("GET /api/v1/logs", hdls.logs.List, auth)
	router.HandleFunc("GET /api/v1/logs/{id}", pathQuery(hdls.log.List, "id"), auth)
	router.HandleFunc("DELETE /api/v1/logs/{id}", pathQuery(hdls.log.Delete, "score"), auth)
	router.HandleFunc("POST /api/v1/logs/{id}/retry", jsonForm(hdls.log.Retry, "id"), auth)
	router.HandleFunc("GET /api/v1/operation-logs", hdls.log.OptLogs, auth, requireMongo(mgo))
	router.HandleFunc("DELETE /api/v1/operation-logs/{id}", pathQuery(hdls.log.DelOptLog, "id"), auth, requireMongo(mgo))
	router.HandleFunc("GET /api/v1/workflow-logs", hdls.log.WorkFlowLogs, auth, requireMongo(mgo))
	router.HandleFunc("GET /api/v1/redis/info/stream", hdls.redisInfo.Info, AuthSSE(mgo, ui, "redis_info"))
	router.HandleFunc("GET /api/v1/redis/monitor/stream", hdls.redisInfo.Monitor, AuthSSE(mgo, ui, "redis_monitor"))
	router.HandleFunc("GET /api/v1/redis/keys", hdls.redisInfo.Keys, auth)
	router.HandleFunc("DELETE /api/v1/redis/keys/{key}", hdls.redisInfo.DeleteKey, auth)
	router.HandleFunc("GET /api/v1/config", hdls.redisInfo.ConfigInfo, auth, requireMongo(mgo))
	router.HandleFunc("PUT /api/v1/config", hdls.redisInfo.Config, auth, requireMongo(mgo))
	router.HandleFunc("POST /api/v1/notifications/test", hdls.login.TestNotify, auth)
	router.HandleFunc("GET /api/v1/clients", hdls.client.List, auth)
	router.HandleFunc("GET /api/v1/dashboard", hdls.dashboard.Total, auth, requireMongo(mgo))
	router.HandleFunc("GET /api/v1/dashboard/metrics", hdls.dashboard.Metrics, auth)
	router.HandleFunc("GET /api/v1/dashboard/stream", hdls.dashboard.Info, AuthSSE(mgo, ui, "dashboard"))
	router.HandleFunc("GET /api/v1/dashboard/pods/stream", hdls.dashboard.Pods, AuthSSE(mgo, ui, "pods"))
	router.HandleFunc("GET /api/v1/nodes", hdls.dashboard.Nodes, auth)

	router.HandleFunc("GET /api/v1/events", hdls.eventLog.List, AuthSSE(mgo, ui, "event_log"), requireMongo(mgo))
	router.HandleFunc("GET /api/v1/events/{id}", pathQuery(hdls.eventLog.Detail, "id"), auth, requireMongo(mgo))
	router.HandleFunc("DELETE /api/v1/events/{id}", jsonForm(hdls.eventLog.Delete, "id"), auth, requireMongo(mgo))
	router.HandleFunc("PATCH /api/v1/events/{id}", jsonForm(hdls.eventLog.Edit, "id"), auth, requireMongo(mgo))
	router.HandleFunc("POST /api/v1/events/{id}/retry", jsonForm(hdls.eventLog.Retry, "id"), auth, requireMongo(mgo))
	router.HandleFunc("GET /api/v1/dead-letters", hdls.dlq.List, auth, requireMongo(mgo))
	router.HandleFunc("DELETE /api/v1/dead-letters/{id}", jsonForm(hdls.dlq.Delete, "id"), auth, requireMongo(mgo))
	router.HandleFunc("POST /api/v1/dead-letters/{id}/retry", jsonForm(hdls.dlq.Retry, "uniqueId"), auth, requireMongo(mgo))
	router.HandleFunc("GET /api/v1/workflows", hdls.workflow.List, auth)
	router.HandleFunc("DELETE /api/v1/workflows/{id}", jsonForm(hdls.workflow.Delete, "id"), auth)
	router.HandleFunc("GET /api/v1/users", hdls.user.List, auth, requireMongo(mgo))
	router.HandleFunc("POST /api/v1/users", hdls.user.Add, auth, requireMongo(mgo))
	router.HandleFunc("PATCH /api/v1/users/{id}", hdls.user.Edit, auth, requireMongo(mgo))
	router.HandleFunc("DELETE /api/v1/users/{id}", hdls.user.Delete, auth, requireMongo(mgo))
	router.HandleFunc("POST /api/v1/users/check-password", hdls.user.Check, auth, requireMongo(mgo))
	router.HandleFunc("GET /api/v1/roles", hdls.role.List, auth, requireMongo(mgo))
	router.HandleFunc("POST /api/v1/roles", hdls.role.Add, auth, requireMongo(mgo))
	router.HandleFunc("PATCH /api/v1/roles/{id}", hdls.role.Edit, auth, requireMongo(mgo))
	router.HandleFunc("DELETE /api/v1/roles/{id}", hdls.role.Delete, auth, requireMongo(mgo))
	router.HandleFunc("GET /api/v1/pods", hdls.pod.List, auth)
	router.HandleFunc("GET /api/v1/mongo", hdls.mongoInfo.Detail, auth, requireMongo(mgo))
	router.HandleFunc("GET /api/v1/tenants", hdls.tenant.List, auth, requireMongo(mgo))
	router.HandleFunc("GET /api/v1/tenants/{id}", hdls.tenant.Get, auth, requireMongo(mgo))
	router.HandleFunc("POST /api/v1/tenants", hdls.tenant.Add, auth, requireMongo(mgo))
	router.HandleFunc("PATCH /api/v1/tenants/{id}", hdls.tenant.Edit, auth, requireMongo(mgo))
	router.HandleFunc("DELETE /api/v1/tenants/{id}", hdls.tenant.Delete, auth, requireMongo(mgo))

	return router
}
