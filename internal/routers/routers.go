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
	r.Get("/schedule", MigrateMiddleWare(NewSchedule(client, prefix).List))
	r.Get("/queue/list", MigrateMiddleWare(NewQueue(client, prefix).List))
	r.Get("/queue/detail", MigrateMiddleWare(NewQueue(client, prefix).Detail))
	r.Get("/logs", MigrateMiddleWare(NewLogs(client, prefix).List))
	r.Get("/log", MigrateMiddleWare(NewLog(client, prefix).List))

	r.Get("/redis", MigrateMiddleWare(NewRedisInfo(client, prefix).Info))
	r.Get("/redis/monitor", MigrateMiddleWare(NewRedisInfo(client, prefix).Monitor))

	r.Post("/login", HeaderRule(NewLogin(client, prefix, ui.Account.UserName, ui.Account.Password, ui.Issuer, ui.Subject, ui.ExpiresAt).Login))
	r.Get("/clients", MigrateMiddleWare(NewClient(client, prefix).List))
	r.Get("/dashboard", MigrateMiddleWare(NewDashboard(client, prefix).Info))
	r.Get("/event_log/list", MigrateMiddleWare(NewEventLog(client, x, prefix).List))
	r.Get("/event_log/detail", MigrateMiddleWare(NewEventLog(client, x, prefix).Detail))
	r.Post("/event_log/delete", MigrateMiddleWare(NewEventLog(client, x, prefix).Delete))
	r.Post("/event_log/edit", MigrateMiddleWare(NewEventLog(client, x, prefix).Edit))
	r.Post("/event_log/retry", MigrateMiddleWare(NewEventLog(client, x, prefix).Retry))

	r.Get("/user/list", MigrateMiddleWare(NewUser(client, prefix).List))
	r.Post("/user/add", MigrateMiddleWare(NewUser(client, prefix).Add))
	r.Post("/user/del", MigrateMiddleWare(NewUser(client, prefix).Delete))
	r.Post("/user/edit", MigrateMiddleWare(NewUser(client, prefix).Edit))

	r.Get("/googleLogin", NewLogin(client, prefix, ui.Account.UserName, ui.Account.Password, ui.Issuer, ui.Subject, ui.ExpiresAt).GoogleLogin)
	r.Get("/callback", NewLogin(client, prefix, ui.Account.UserName, ui.Account.Password, ui.Issuer, ui.Subject, ui.ExpiresAt).GoogleCallBack)

	r.Get("/dlq/list", MigrateMiddleWare(NewDlq(client, prefix).List))
	return r
}
