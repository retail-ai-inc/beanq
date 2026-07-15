package routers

import (
	"net/http"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
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

	page, err := parsePage(r)
	if err != nil {
		writeBadRequest(w, err)
		return
	}

	data, total, err := t.mgo.Roles(r.Context(), nil, page.Page, page.PageSize)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = map[string]any{"data": data, "total": total, "cursor": page.Page}
	_ = res.Json(w, http.StatusOK)
}

func (t *Role) Add(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	var input struct {
		Name  string `json:"name"`
		Roles []int  `json:"roles"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeBadRequest(w, err)
		return
	}

	if input.Name == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "missing name"
		_ = res.Json(w, http.StatusBadRequest)
		return
	}
	if err := t.mgo.AddRole(r.Context(), &bmongo.Role{
		Name: input.Name, Roles: input.Roles,
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

	id := r.PathValue("id")

	if id == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "id is required"
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	if _, err := t.mgo.DeleteRole(r.Context(), id); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	_ = res.Json(w, http.StatusOK)
}

func (t *Role) Edit(w http.ResponseWriter, r *http.Request) {

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
		Roles  []int  `json:"roles"`
		Detail string `json:"detail"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeBadRequest(w, err)
		return
	}

	if _, err := t.mgo.EditRole(r.Context(), id, map[string]any{"roles": input.Roles, "detail": input.Detail}); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	_ = res.Json(w, http.StatusOK)
}
