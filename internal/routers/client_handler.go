package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"net/http"
)

type Client struct {
	client redis.UniversalClient
	prefix string
}

func NewClient(client redis.UniversalClient, prefix string) *Client {
	return &Client{client: client, prefix: prefix}
}

func (t *Client) List(ctx *bwebframework.BeanContext) error {

	r := ctx.Request
	w := ctx.Writer

	result, cancel := response.Get()
	defer cancel()
	nodeId := r.Header.Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	data, err := client.ClientList(r.Context())
	if err != nil {
		result.Code = "1001"
		result.Msg = err.Error()
		return result.Json(w, http.StatusInternalServerError)

	}
	result.Data = data
	return result.Json(w, http.StatusOK)

}
