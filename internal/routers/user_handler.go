package routers

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/email"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"strings"
)

type User struct {
	Account string `json:"account"`
	client  redis.UniversalClient
	prefix  string
}

func NewUser(client redis.UniversalClient, prefix string) *User {
	return &User{client: client, prefix: prefix}
}

func (t *User) List(ctx *bwebframework.BeanContext) error {

	res, cancel := response.Get()
	defer cancel()

	r := ctx.Request
	w := ctx.Writer

	nodeId := r.Header.Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	pattern := strings.Join([]string{viper.GetString("redis.prefix"), "users:*"}, ":")
	keys, err := client.Keys(r.Context(), pattern)

	if err != nil {
		res.Code = berror.InternalServerErrorMsg
		res.Msg = err.Error()
		return res.Json(w, http.StatusOK)
	}

	data := make([]any, 0)
	for _, key := range keys {

		r, err := HGetAll(r.Context(), t.client, key)
		if err != nil {
			fmt.Printf("hget err:%+v \n", err)
			continue
		}

		data = append(data, r)
	}
	res.Data = data
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

	key := strings.Join([]string{viper.GetString("redis.prefix"), "users", account}, ":")
	data := make(map[string]any, 0)
	data["account"] = account
	data["password"] = password
	data["type"] = typ
	data["active"] = active
	data["detail"] = detail

	if err := HSet(r.Context(), t.client, key, data); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusOK)

	}
	go func(ctx2 context.Context) {

		code, body, header, err := email.DefaultSend(ctx2, "BeanqUI Manager", account, &email.EmbedData{
			Title: "Active Email",
			Name:  account,
			Link:  "", // website url
		})
		if err != nil {
			log.Printf("Send Email Error:%+v \n", err)
		}
		if code != 200 {
			log.Printf("Code:%+v,Body:%+v,Header:%+v \n", code, body, header)
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

	account := ctx.Request.PostFormValue("account")

	if account == "" {
		res.Code = berror.MissParameterMsg
		res.Msg = "missing account field"
		return res.Json(ctx.Writer, http.StatusOK)
	}

	key := strings.Join([]string{viper.GetString("redis.prefix"), "users", account}, ":")
	if err := Del(ctx.Request.Context(), t.client, key); err != nil {
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

	account := r.FormValue("account")
	password := r.FormValue("password")
	active := r.FormValue("active")
	typ := r.FormValue("type")
	detail := r.FormValue("detail")

	key := strings.Join([]string{viper.GetString("redis.prefix"), "users", account}, ":")
	var data = map[string]any{
		"account":  account,
		"password": password,
		"active":   active,
		"detail":   detail,
		"type":     typ,
	}
	if err := HSet(r.Context(), t.client, key, data); err != nil {
		res.Code = berror.InternalServerErrorMsg
		res.Msg = err.Error()
		return res.Json(w, http.StatusOK)

	}
	return res.Json(w, http.StatusOK)

}
