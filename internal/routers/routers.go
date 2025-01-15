package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/mongox"
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

func NewRouters(r *bwebframework.Router, client redis.UniversalClient, x *mongox.MongoX, prefix string, ui Ui) *bwebframework.Router {

	r.Get("/ping", HeaderRule(func(ctx *bwebframework.BeanContext) error {

		ctx.Writer.WriteHeader(http.StatusOK)
		_, _ = ctx.Writer.Write([]byte("pong"))
		return nil
	}))
	r.Get("/schedule", MigrateMiddleWare(NewSchedule(client, prefix).List, client, x, prefix))
	r.Get("/queue/list", MigrateMiddleWare(NewQueue(client, prefix).List, client, x, prefix))
	r.Get("/queue/detail", MigrateMiddleWare(NewQueue(client, prefix).Detail, client, x, prefix))
	r.Get("/logs", MigrateMiddleWare(NewLogs(client, prefix).List, client, x, prefix))
	r.Get("/log", MigrateMiddleWare(NewLog(client, x, prefix).List, client, x, prefix))
	r.Get("/log/opt_log", MigrateMiddleWare(NewLog(client, x, prefix).OptLogs, client, x, prefix))
	r.Delete("/log/opt_log", MigrateMiddleWare(NewLog(client, x, prefix).DelOptLog, client, x, prefix))

	r.Get("/log/workflow_log", MigrateMiddleWare(NewLog(client, x, prefix).WorkFlowLogs, client, x, prefix))

	r.Get("/redis", MigrateMiddleWare(NewRedisInfo(client, prefix).Info, client, x, prefix))
	r.Get("/redis/monitor", MigrateMiddleWare(NewRedisInfo(client, prefix).Monitor, client, x, prefix))

	r.Post("/login", HeaderRule(NewLogin(client, prefix, ui.Account.UserName, ui.Account.Password, ui.Issuer, ui.Subject, ui.ExpiresAt).Login))
	r.Get("/clients", MigrateMiddleWare(NewClient(client, prefix).List, client, x, prefix))

	r.Get("/dashboard", MigrateMiddleWare(NewDashboard(client, x, prefix).Info, client, x, prefix))
	r.Get("/nodes", MigrateMiddleWare(NewDashboard(client, x, prefix).Nodes, client, x, prefix))

	r.Get("/event_log/list", MigrateMiddleWare(NewEventLog(client, x, prefix).List, client, x, prefix))
	r.Get("/event_log/detail", MigrateMiddleWare(NewEventLog(client, x, prefix).Detail, client, x, prefix))
	r.Post("/event_log/delete", MigrateMiddleWare(NewEventLog(client, x, prefix).Delete, client, x, prefix))
	r.Post("/event_log/edit", MigrateMiddleWare(NewEventLog(client, x, prefix).Edit, client, x, prefix))
	r.Post("/event_log/retry", MigrateMiddleWare(NewEventLog(client, x, prefix).Retry, client, x, prefix))

	r.Get("/user/list", MigrateMiddleWare(NewUser(client, prefix).List, client, x, prefix))
	r.Post("/user/add", MigrateMiddleWare(NewUser(client, prefix).Add, client, x, prefix))
	r.Post("/user/del", MigrateMiddleWare(NewUser(client, prefix).Delete, client, x, prefix))
	r.Post("/user/edit", MigrateMiddleWare(NewUser(client, prefix).Edit, client, x, prefix))

	r.Get("/googleLogin", NewLogin(client, prefix, ui.Account.UserName, ui.Account.Password, ui.Issuer, ui.Subject, ui.ExpiresAt).GoogleLogin)
	r.Get("/callback", NewLogin(client, prefix, ui.Account.UserName, ui.Account.Password, ui.Issuer, ui.Subject, ui.ExpiresAt).GoogleCallBack)

	r.Get("/dlq/list", MigrateMiddleWare(NewDlq(client, prefix).List, client, x, prefix))
	return r
}
