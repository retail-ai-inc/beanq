package routers

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/email"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"net/http"
)

type User struct {
	Account string `json:"account"`
	client  redis.UniversalClient
	mgo     *bmongo.BMongo
	prefix  string
}

func NewUser(client redis.UniversalClient, x *bmongo.BMongo, prefix string) *User {
	return &User{client: client, mgo: x, prefix: prefix}
}

func (t *User) List(ctx *bwebframework.BeanContext) error {

	res, cancel := response.Get()
	defer cancel()

	r := ctx.Request
	w := ctx.Writer

	page := cast.ToInt64(r.URL.Query().Get("page"))
	pageSize := cast.ToInt64(r.URL.Query().Get("pageSize"))
	account := r.URL.Query().Get("account")

	filter := bson.M{}
	if account != "" {
		filter["account"] = bson.M{
			"$regex":   account,
			"$options": "i",
		}
	}

	data, total, err := t.mgo.UserLogs(r.Context(), filter, page, pageSize)

	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusInternalServerError)
	}
	res.Data = map[string]any{"data": data, "total": total, "cursor": page}
	return res.Json(w, http.StatusOK)
}

func (t *User) Add(ctx *bwebframework.BeanContext) error {
	res, cancel := response.Get()
	defer cancel()

	r := ctx.Request
	w := ctx.Writer

	account := r.PostFormValue("account")
	password := r.PostFormValue("password")
	typ := r.PostFormValue("type")
	active := r.PostFormValue("active")
	detail := r.PostFormValue("detail")

	if account == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "missing account"
		return res.Json(w, http.StatusOK)
	}

	if err := t.mgo.AddUser(r.Context(), &bmongo.User{
		Account:  account,
		Password: password,
		Type:     typ,
		Active:   cast.ToInt32(active),
		Detail:   detail,
	}); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusInternalServerError)
	}

	go func(ctx2 context.Context) {

		client, err := email.NewEmail(ctx2, viper.GetString("ui.sendGrid.key"))
		if err != nil {
			log.Printf("Email Error:%+v \n", err)
		}
		client.From("BeanqUI Manager")
		client.To(account)
		client.Subject("BeanqUI Manager")
		_ = client.Body("Active Email", account, "")
		if err := client.Send(); err != nil {
			log.Printf("Send Email Error:%+v \n", err)
		}

	}(r.Context())
	return res.Json(w, http.StatusOK)
}

type UserInfo struct {
	Account string `json:"account"`
}

func (t *User) Delete(ctx *bwebframework.BeanContext) error {

	res, cancel := response.Get()
	defer cancel()

	id := ctx.Request.PostFormValue("id")

	if id == "" {
		res.Code = berror.MissParameterMsg
		res.Msg = "missing account field"
		return res.Json(ctx.Writer, http.StatusOK)
	}

	if _, err := t.mgo.DeleteUser(ctx.Request.Context(), id); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(ctx.Writer, http.StatusOK)
	}

	return res.Json(ctx.Writer, http.StatusOK)

}

func (t *User) Edit(ctx *bwebframework.BeanContext) error {

	res, cancel := response.Get()
	defer cancel()

	r := ctx.Request
	w := ctx.Writer

	id := r.FormValue("_id")
	if id == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "ID can't be empty"
		return res.Json(w, http.StatusBadRequest)
	}
	account := r.FormValue("account")
	password := r.FormValue("password")
	active := r.FormValue("active")
	typ := r.FormValue("type")
	detail := r.FormValue("detail")

	if _, err := t.mgo.EditUser(r.Context(), id, map[string]any{"account": account, "password": password, "active": active, "type": typ, "detail": detail}); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusInternalServerError)
	}
	return res.Json(w, http.StatusOK)

}
