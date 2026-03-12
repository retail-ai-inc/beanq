package routers

import (
	"net/http"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
)

type MongoInfo struct {
	mgo *bmongo.BMongo
}

func NewMongoInfo(mgo *bmongo.BMongo) *MongoInfo {
	return &MongoInfo{mgo: mgo}
}

func (m *MongoInfo) Detail(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()

	stats, err := m.mgo.MongoDetail(r.Context())
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	result.Data = stats
	_ = result.Json(w, http.StatusOK)

}
