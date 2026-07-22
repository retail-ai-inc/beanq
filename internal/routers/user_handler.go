package routers

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"github.com/retail-ai-inc/beanq/v4/helper/ui"
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

	page, err := parsePage(r)
	if err != nil {
		writeBadRequest(w, err)
		return
	}
	account := r.URL.Query().Get("account")
	if len(account) > 128 {
		writeBadRequest(w, errors.New("account filter must not exceed 128 characters"))
		return
	}

	filter := bson.M{}
	if account != "" {
		filter["account"] = bson.M{
			"$regex":   regexp.QuoteMeta(account),
			"$options": "i",
		}
	}

	data, total, err := t.mgo.UserLogs(r.Context(), filter, page.Page, page.PageSize)

	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = map[string]any{"data": data, "total": total, "cursor": page.Page}
	_ = res.Json(w, http.StatusOK)

}

func (t *User) Add(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	var input struct {
		Account  string `json:"account"`
		Password string `json:"password"`
		Type     string `json:"type"`
		Active   int32  `json:"active"`
		Detail   string `json:"detail"`
		RoleID   string `json:"roleId"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeBadRequest(w, err)
		return
	}
	input.Account = strings.TrimSpace(input.Account)

	if input.Account == "" || input.Password == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "account and password are required"
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	if err := t.mgo.AddUser(r.Context(), &bmongo.User{
		Account: input.Account, Password: input.Password, Type: input.Type,
		Active: input.Active, Detail: input.Detail, RoleId: input.RoleID,
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

	id := r.PathValue("id")

	if id == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "id is required"
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	count, err := t.mgo.DeleteUser(r.Context(), id)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	if count == 0 {
		writeAPIError(w, http.StatusNotFound, berror.MissParameterCode, "user not found")
		return
	}
	_ = res.Json(w, http.StatusOK)
}

func (t *User) Edit(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "ID can't be empty"
		_ = res.Json(w, http.StatusBadRequest)
		return
	}
	var input struct {
		Password *string `json:"password"`
		Active   *int32  `json:"active"`
		Type     *string `json:"type"`
		Detail   *string `json:"detail"`
		RoleID   *string `json:"roleId"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeBadRequest(w, err)
		return
	}
	updates := map[string]any{"password": input.Password, "active": input.Active, "type": input.Type, "detail": input.Detail, "roleId": input.RoleID}
	count, err := t.mgo.EditUser(r.Context(), id, updates)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	if count == 0 {
		writeAPIError(w, http.StatusNotFound, berror.MissParameterCode, "user not found")
		return
	}
	_ = res.Json(w, http.StatusOK)
}

func (t *User) Check(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	username, ok := r.Context().Value(UserName).(string)
	if !ok || username == "" {
		writeAPIError(w, http.StatusUnauthorized, berror.AuthExpireCode, "unauthorized")
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeBadRequest(w, err)
		return
	}
	pwd := input.Password

	if username == t.ui.Root.UserName && pwd == t.ui.Root.Password {
		_ = res.Json(w, http.StatusOK)
		return
	}

	if _, err := t.mgo.CheckUser(r.Context(), username, pwd); err == nil {
		_ = res.Json(w, http.StatusOK)
		return
	}
	res.Code = berror.AuthExpireCode
	res.Msg = "Unauthorized"
	res.Data = "Unauthorized"
	_ = res.Json(w, http.StatusUnauthorized)
}
