package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"net/http"
	"time"
)

type Ui struct {
	Account struct {
		UserName string `json:"username"`
		Password string `json:"password"`
	} `json:"account"`
	Issuer    string        `json:"issuer"`
	Subject   string        `json:"subject"`
	JwtKey    string        `json:"jwtKey"`
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
	role      *Role
}

func NewRouters(r *bwebframework.Router, client redis.UniversalClient, mgo *bmongo.BMongo, prefix string, ui Ui) *bwebframework.Router {

	hdls := Handles{
		schedule:  NewSchedule(client, prefix),
		queue:     NewQueue(client, prefix),
		logs:      NewLogs(client, prefix),
		log:       NewLog(client, mgo, prefix),
		redisInfo: NewRedisInfo(client, prefix),
		login:     NewLogin(client, mgo, prefix, ui.Account.UserName, ui.Account.Password, ui.Issuer, ui.Subject, ui.ExpiresAt),
		client:    NewClient(client, prefix),
		dashboard: NewDashboard(client, mgo, prefix),
		eventLog:  NewEventLog(client, mgo, prefix),
		user:      NewUser(client, mgo, prefix),
		dlq:       NewDlq(client, prefix),
		role:      NewRole(mgo),
	}

	r.Get("/ping", HeaderRule(func(ctx *bwebframework.BeanContext) error {

		ctx.Writer.WriteHeader(http.StatusOK)
		_, _ = ctx.Writer.Write([]byte("pong"))
		return nil
	}))
	r.Get("/schedule", MigrateMiddleWare(hdls.schedule.List, client, mgo, prefix))
	r.Get("/queue/list", MigrateMiddleWare(hdls.queue.List, client, mgo, prefix))
	r.Get("/queue/detail", MigrateMiddleWare(hdls.queue.Detail, client, mgo, prefix))
	r.Get("/logs", MigrateMiddleWare(hdls.logs.List, client, mgo, prefix))
	r.Get("/log", MigrateMiddleWare(hdls.log.List, client, mgo, prefix))
	r.Get("/log/opt_log", MigrateMiddleWare(hdls.log.OptLogs, client, mgo, prefix))
	r.Delete("/log/opt_log", MigrateMiddleWare(hdls.log.DelOptLog, client, mgo, prefix))

	r.Get("/log/workflow_log", MigrateMiddleWare(hdls.log.WorkFlowLogs, client, mgo, prefix))

	r.Get("/redis", MigrateMiddleWare(hdls.redisInfo.Info, client, mgo, prefix))
	r.Get("/redis/monitor", MigrateMiddleWare(hdls.redisInfo.Monitor, client, mgo, prefix))

	r.Post("/login", HeaderRule(hdls.login.Login))
	r.Get("/clients", MigrateMiddleWare(hdls.client.List, client, mgo, prefix))

	r.Get("/dashboard", MigrateMiddleWare(hdls.dashboard.Info, client, mgo, prefix))
	r.Get("/nodes", MigrateMiddleWare(hdls.dashboard.Nodes, client, mgo, prefix))

	r.Get("/event_log/list", MigrateMiddleWare(hdls.eventLog.List, client, mgo, prefix))
	r.Get("/event_log/detail", MigrateMiddleWare(hdls.eventLog.Detail, client, mgo, prefix))
	r.Post("/event_log/delete", MigrateMiddleWare(hdls.eventLog.Delete, client, mgo, prefix))
	r.Post("/event_log/edit", MigrateMiddleWare(hdls.eventLog.Edit, client, mgo, prefix))
	r.Post("/event_log/retry", MigrateMiddleWare(hdls.eventLog.Retry, client, mgo, prefix))

	r.Get("/user/list", MigrateMiddleWare(hdls.user.List, client, mgo, prefix))
	r.Post("/user/add", MigrateMiddleWare(hdls.user.Add, client, mgo, prefix))
	r.Post("/user/del", MigrateMiddleWare(hdls.user.Delete, client, mgo, prefix))
	r.Post("/user/edit", MigrateMiddleWare(hdls.user.Edit, client, mgo, prefix))

	r.Get("/role/list", MigrateMiddleWare(hdls.role.List, nil, mgo, ""))
	r.Post("/role/add", MigrateMiddleWare(hdls.role.Add, nil, mgo, ""))
	r.Post("/role/delete", MigrateMiddleWare(hdls.role.Delete, nil, mgo, ""))
	r.Post("/role/edit", MigrateMiddleWare(hdls.role.Edit, nil, mgo, ""))

	r.Get("/googleLogin", hdls.login.GoogleLogin)
	r.Get("/callback", hdls.login.GoogleCallBack)

	r.Get("/dlq/list", MigrateMiddleWare(hdls.dlq.List, client, mgo, prefix))
	return r
}
