package routers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
)

type Tenants struct {
	mgo *bmongo.BMongo
}

func NewTenants(mgo *bmongo.BMongo) *Tenants {
	return &Tenants{mgo: mgo}
}

func (t *Tenants) List(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	data, total, err := t.mgo.TenantsList(r.Context(), 0, 10)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = map[string]any{
		"rows":  data,
		"total": total,
	}
	_ = res.Json(w, http.StatusOK)
	return

}

func (t *Tenants) Add(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	tenant := bmongo.Tenants{}
	if err := json.NewDecoder(r.Body).Decode(&tenant); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}

	tenant.CreateAt = time.Now()

	id, err := t.mgo.TenantsAdd(r.Context(), &tenant)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = map[string]string{"id": id}
	_ = res.Json(w, http.StatusOK)
	return

}

func (t *Tenants) Delete(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		res.Code = berror.InternalServerErrorCode
		res.Msg = "id is required"
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	if err := t.mgo.TenantsDelete(r.Context(), id); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	_ = res.Json(w, http.StatusOK)
	return
}

func (t *Tenants) Edit(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()
	id := r.PathValue("id")
	if id == "" {
		res.Code = berror.InternalServerErrorCode
		res.Msg = "id is required"
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}

	tenant := bmongo.Tenants{}
	if err := json.NewDecoder(r.Body).Decode(&tenant); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	if err := t.mgo.TenantsEdit(r.Context(), id, &tenant); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	_ = res.Json(w, http.StatusOK)
	return
}

func (t *Tenants) Get(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()
	id := r.PathValue("id")
	if id == "" {
		res.Code = berror.InternalServerErrorCode
		res.Msg = "id required"
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	tenant, err := t.mgo.TenantsInfo(r.Context(), id)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	res.Data = tenant
	_ = res.Json(w, http.StatusOK)
	return
}
