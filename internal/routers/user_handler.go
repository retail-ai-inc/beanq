package routers

import (
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"github.com/retail-ai-inc/beanq/v4/helper/ui"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
)

type User struct {
	Account string `json:"account"`
	client  redis.UniversalClient
	mgo     *bmongo.BMongo
	prefix  string
	ui      ui.Ui
}

func NewUser(client redis.UniversalClient, x *bmongo.BMongo, prefix string, ui ui.Ui) *User {
	return &User{client: client, mgo: x, prefix: prefix, ui: ui}
}

func (t *User) List(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

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
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = map[string]any{"data": data, "total": total, "cursor": page}
	_ = res.Json(w, http.StatusOK)

}

func (t *User) Add(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	account := r.PostFormValue("account")
	password := r.PostFormValue("password")
	typ := r.PostFormValue("type")
	active := r.PostFormValue("active")
	detail := r.PostFormValue("detail")
	roleId := r.PostFormValue("roleId")

	if account == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "missing account"
		_ = res.Json(w, http.StatusOK)
		return
	}

	if err := t.mgo.AddUser(r.Context(), &bmongo.User{
		Account:  account,
		Password: password,
		Type:     typ,
		Active:   cast.ToInt32(active),
		Detail:   detail,
		RoleId:   roleId,
	}); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	//todo send email will use another way
	//

	_ = res.Json(w, http.StatusOK)
}

type UserInfo struct {
	Account string `json:"account"`
}

func (t *User) Delete(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	id := r.PostFormValue("id")

	if id == "" {
		res.Code = berror.MissParameterMsg
		res.Msg = "missing account field"
		_ = res.Json(w, http.StatusOK)
		return
	}

	if _, err := t.mgo.DeleteUser(r.Context(), id); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusOK)
		return
	}
	_ = res.Json(w, http.StatusOK)
}

func (t *User) Edit(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	id := r.FormValue("_id")
	if id == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "ID can't be empty"
		_ = res.Json(w, http.StatusBadRequest)
		return
	}
	account := r.FormValue("account")
	password := r.FormValue("password")
	active := r.FormValue("active")
	typ := r.FormValue("type")
	detail := r.FormValue("detail")
	roleId := r.FormValue("roleId")

	if _, err := t.mgo.EditUser(r.Context(), id, map[string]any{"account": account, "password": password, "active": active, "type": typ, "detail": detail, "roleId": roleId}); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	_ = res.Json(w, http.StatusOK)
}

func (t *User) Check(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	username := r.Context().Value(UserName)
	pwd := r.FormValue("password")

	if username == t.ui.Root.UserName && pwd == t.ui.Root.Password {
		_ = res.Json(w, http.StatusOK)
		return
	}

	if _, err := t.mgo.CheckUser(r.Context(), username.(string), pwd); err == nil {
		_ = res.Json(w, http.StatusOK)
		return
	}
	res.Code = berror.SuccessCode
	res.Msg = "Unauthorized"
	res.Data = "Unauthorized"
	_ = res.Json(w, http.StatusOK)

}
