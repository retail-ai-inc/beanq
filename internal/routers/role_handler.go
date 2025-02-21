package routers

import (
	"encoding/json"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
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

func (t *Role) List(ctx *bwebframework.BeanContext) error {
	res, cancel := response.Get()
	defer cancel()

	r := ctx.Request
	w := ctx.Writer

	page := cast.ToInt64(r.URL.Query().Get("page"))
	pageSize := cast.ToInt64(r.URL.Query().Get("pageSize"))

	data, total, err := t.mgo.Roles(r.Context(), nil, page, pageSize)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusInternalServerError)
	}
	res.Data = map[string]any{"data": data, "total": total, "cursor": page}
	return res.Json(w, http.StatusOK)
}

func (t *Role) Add(ctx *bwebframework.BeanContext) error {
	res, cancel := response.Get()
	defer cancel()

	r := ctx.Request
	w := ctx.Writer

	name := r.PostFormValue("name")
	roles := r.PostFormValue("roles")

	if name == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "missing name"
		return res.Json(w, http.StatusOK)
	}
	role := make([]int, 0)
	if err := json.Unmarshal([]byte(roles), &role); err != nil {
		res.Code = berror.TypeErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusInternalServerError)
	}
	if err := t.mgo.AddRole(r.Context(), &bmongo.Role{
		Name:  name,
		Roles: role,
	}); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusInternalServerError)
	}
	return res.Json(w, http.StatusOK)
}

func (t *Role) Delete(ctx *bwebframework.BeanContext) error {
	res, cancel := response.Get()
	defer cancel()

	id := ctx.Request.PostFormValue("id")

	if id == "" {
		res.Code = berror.MissParameterMsg
		res.Msg = "missing account field"
		return res.Json(ctx.Writer, http.StatusOK)
	}

	if _, err := t.mgo.DeleteRole(ctx.Request.Context(), id); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(ctx.Writer, http.StatusOK)
	}

	return res.Json(ctx.Writer, http.StatusOK)
}

func (t *Role) Edit(ctx *bwebframework.BeanContext) error {

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
	roles := r.PostFormValue("roles")
	role := make([]int, 0)
	if err := json.Unmarshal([]byte(roles), &role); err != nil {
		res.Code = berror.TypeErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusInternalServerError)
	}

	if _, err := t.mgo.EditRole(r.Context(), id, map[string]any{"roles": role}); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusInternalServerError)
	}
	return res.Json(w, http.StatusOK)
}
