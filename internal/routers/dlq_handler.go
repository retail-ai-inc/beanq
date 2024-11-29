package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/response"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"net/http"
	"strings"
)

type Dlq struct {
	client redis.UniversalClient
	prefix string
}

func NewDlq(client redis.UniversalClient, prefix string) *Dlq {
	return &Dlq{client: client, prefix: prefix}
}

func (t *Dlq) List(ctx *bwebframework.BeanContext) error {

	w := ctx.Writer
	r := ctx.Request

	res, cancel := response.Get()
	defer cancel()

	stream := strings.Join([]string{viper.GetString("redis.prefix"), "beanq-logic-log"}, ":")

	msgs, err := XRevRange(r.Context(), t.client, stream, "+", "-")
	if err != nil {
		res.Code = berror.InternalServerErrorMsg
		res.Msg = err.Error()
		return res.Json(w, http.StatusOK)

	}
	data := make([]map[string]any, 0)
	for _, msg := range msgs {
		val := msg.Values
		if v, ok := val["pendingRetry"]; ok {
			if cast.ToInt(v) > 0 {
				data = append(data, val)
			}
		}
	}
	res.Data = data
	return res.Json(w, http.StatusOK)

}
