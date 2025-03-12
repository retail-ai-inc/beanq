package routers

import (
	"encoding/json"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/spf13/cast"
	"net/http"
)

type Role struct {
	mgo *bmongo.BMongo
}

func NewRole(mongo *bmongo.BMongo) *Role {
	return &Role{mgo: mongo}
}

func (t *Role) List(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	page := cast.ToInt64(r.URL.Query().Get("page"))
	pageSize := cast.ToInt64(r.URL.Query().Get("pageSize"))

	data, total, err := t.mgo.Roles(r.Context(), nil, page, pageSize)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = map[string]any{"data": data, "total": total, "cursor": page}
	_ = res.Json(w, http.StatusOK)
}

func (t *Role) Add(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	name := r.PostFormValue("name")
	roles := r.PostFormValue("roles")

	if name == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "missing name"
		_ = res.Json(w, http.StatusOK)
		return
	}
	role := make([]int, 0)
	if err := json.Unmarshal([]byte(roles), &role); err != nil {
		res.Code = berror.TypeErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	if err := t.mgo.AddRole(r.Context(), &bmongo.Role{
		Name:  name,
		Roles: role,
	}); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	_ = res.Json(w, http.StatusOK)
}

func (t *Role) Delete(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	id := r.PostFormValue("id")

	if id == "" {
		res.Code = berror.MissParameterMsg
		res.Msg = "missing account field"
		_ = res.Json(w, http.StatusOK)
		return
	}

	if _, err := t.mgo.DeleteRole(r.Context(), id); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusOK)
		return
	}
	_ = res.Json(w, http.StatusOK)
}

func (t *Role) Edit(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	id := r.FormValue("_id")
	if id == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "ID can't be empty"
		_ = res.Json(w, http.StatusBadRequest)
		return
	}
	roles := r.PostFormValue("roles")
	role := make([]int, 0)
	if err := json.Unmarshal([]byte(roles), &role); err != nil {
		res.Code = berror.TypeErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}

	if _, err := t.mgo.EditRole(r.Context(), id, map[string]any{"roles": role}); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	_ = res.Json(w, http.StatusOK)
}
