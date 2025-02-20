package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"net/http"
	"time"
)

type Ui struct {
	Stmt struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
	}
	GoogleAuth struct {
		ClientId     string
		ClientSecret string
		CallbackUrl  string
	}
	SendGrid struct {
		Key         string
		FromName    string
		FromAddress string
	}
	Root struct {
		UserName string `json:"username"`
		Password string `json:"password"`
	} `json:"root"`
	On        bool          `json:"on"`
	Issuer    string        `json:"issuer"`
	Subject   string        `json:"subject"`
	JwtKey    string        `json:"jwtKey"`
	Port      string        `json:"port"`
	ExpiresAt time.Duration `json:"expiresAt"`
}

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
}

func NewRouters(r *bwebframework.Router, client redis.UniversalClient, mgo *bmongo.BMongo, prefix string, ui Ui) *bwebframework.Router {

	hdls := Handles{
		schedule:  NewSchedule(client, prefix),
		queue:     NewQueue(client, prefix),
		logs:      NewLogs(client, prefix),
		log:       NewLog(client, mgo, prefix),
		redisInfo: NewRedisInfo(client, prefix),
		login:     NewLogin(client, mgo, prefix, ui),
		client:    NewClient(client, prefix),
		dashboard: NewDashboard(client, mgo, prefix),
		eventLog:  NewEventLog(client, mgo, prefix),
		user:      NewUser(client, mgo, prefix),
		dlq:       NewDlq(client, mgo, prefix),
		workflow:  NewWorkFlow(client, mgo, prefix),
		role:      NewRole(mgo),
	}

	r.Get("/ping", HeaderRule(func(ctx *bwebframework.BeanContext) error {

		ctx.Writer.WriteHeader(http.StatusOK)
		_, _ = ctx.Writer.Write([]byte("pong"))
		return nil
	}))
	r.Get("/schedule", MigrateMiddleWare(hdls.schedule.List, client, mgo, prefix, ui))
	r.Get("/queue/list", MigrateMiddleWare(hdls.queue.List, client, mgo, prefix, ui))
	r.Get("/queue/detail", MigrateMiddleWare(hdls.queue.Detail, client, mgo, prefix, ui))
	r.Get("/logs", MigrateMiddleWare(hdls.logs.List, client, mgo, prefix, ui))
	r.Get("/log", MigrateMiddleWare(hdls.log.List, client, mgo, prefix, ui))
	r.Get("/log/opt_log", MigrateMiddleWare(hdls.log.OptLogs, client, mgo, prefix, ui))
	r.Delete("/log/opt_log", MigrateMiddleWare(hdls.log.DelOptLog, client, mgo, prefix, ui))

	r.Get("/log/workflow_log", MigrateMiddleWare(hdls.log.WorkFlowLogs, client, mgo, prefix, ui))

	r.Get("/redis", MigrateMiddleWare(hdls.redisInfo.Info, client, mgo, prefix, ui))
	r.Get("/redis/monitor", MigrateMiddleWare(hdls.redisInfo.Monitor, client, mgo, prefix, ui))

	r.Post("/login", HeaderRule(hdls.login.Login))
	r.Get("/clients", MigrateMiddleWare(hdls.client.List, client, mgo, prefix, ui))

	r.Get("/dashboard", MigrateMiddleWare(hdls.dashboard.Info, client, mgo, prefix, ui))
	r.Get("/nodes", MigrateMiddleWare(hdls.dashboard.Nodes, client, mgo, prefix, ui))

	r.Get("/event_log/list", MigrateMiddleWare(hdls.eventLog.List, client, mgo, prefix, ui))
	r.Get("/event_log/detail", MigrateMiddleWare(hdls.eventLog.Detail, client, mgo, prefix, ui))
	r.Post("/event_log/delete", MigrateMiddleWare(hdls.eventLog.Delete, client, mgo, prefix, ui))
	r.Post("/event_log/edit", MigrateMiddleWare(hdls.eventLog.Edit, client, mgo, prefix, ui))
	r.Post("/event_log/retry", MigrateMiddleWare(hdls.eventLog.Retry, client, mgo, prefix, ui))

	r.Get("/user/list", MigrateMiddleWare(hdls.user.List, client, mgo, prefix, ui))
	r.Post("/user/add", MigrateMiddleWare(hdls.user.Add, client, mgo, prefix, ui))
	r.Post("/user/del", MigrateMiddleWare(hdls.user.Delete, client, mgo, prefix, ui))
	r.Post("/user/edit", MigrateMiddleWare(hdls.user.Edit, client, mgo, prefix, ui))

	r.Get("/role/list", MigrateMiddleWare(hdls.role.List, nil, mgo, "", ui))
	r.Post("/role/add", MigrateMiddleWare(hdls.role.Add, nil, mgo, "", ui))
	r.Post("/role/delete", MigrateMiddleWare(hdls.role.Delete, nil, mgo, "", ui))
	r.Post("/role/edit", MigrateMiddleWare(hdls.role.Edit, nil, mgo, "", ui))

	r.Get("/googleLogin", hdls.login.GoogleLogin)
	r.Get("/callback", hdls.login.GoogleCallBack)

	r.Get("/dlq/list", MigrateMiddleWare(hdls.dlq.List, client, mgo, prefix, ui))
	r.Post("/dlq/retry", MigrateMiddleWare(hdls.dlq.Retry, client, mgo, prefix, ui))
	r.Post("/dlq/delete", MigrateMiddleWare(hdls.dlq.Delete, client, mgo, prefix, ui))
	r.Get("/workflow/list", MigrateMiddleWare(hdls.workflow.List, client, mgo, prefix, ui))
	r.Post("/workflow/delete", MigrateMiddleWare(hdls.workflow.Delete, client, mgo, prefix, ui))
	return r
}
