package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"net/http"
)

type Schedule struct {
	client redis.UniversalClient
	prefix string
}

func NewSchedule(client redis.UniversalClient, prefix string) *Schedule {
	return &Schedule{client: client, prefix: prefix}
}

func (t *Schedule) List(ctx *bwebframework.BeanContext) error {

	result, cancel := response.Get()
	defer cancel()

	bt, err := QueueInfo(ctx.Request.Context(), t.client, t.prefix)

	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()

		return result.Json(ctx.Writer, http.StatusInternalServerError)
	}
	result.Data = bt
	return result.Json(ctx.Writer, http.StatusOK)
}
