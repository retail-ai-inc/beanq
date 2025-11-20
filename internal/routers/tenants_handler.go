package routers

import (
	"net/http"

	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
)

type Tenants struct {
	mgo *bmongo.BMongo
}

func NewTenants(mgo *bmongo.BMongo) *Tenants {
	return &Tenants{mgo: mgo}
}

func (t *Tenants) List(w http.ResponseWriter, r *http.Request) {
}

func (t *Tenants) Add(w http.ResponseWriter, r *http.Request) {
}

func (t *Tenants) Delete(w http.ResponseWriter, r *http.Request) {
}

func (t *Tenants) Edit(w http.ResponseWriter, r *http.Request) {}

func (t *Tenants) Get(w http.ResponseWriter, r *http.Request) {}
