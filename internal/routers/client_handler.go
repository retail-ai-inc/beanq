package routers

import (
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
)

type Client struct {
	client redis.UniversalClient
	prefix string
}

func NewClient(client redis.UniversalClient, prefix string) *Client {
	return &Client{client: client, prefix: prefix}
}

func (t *Client) List(w http.ResponseWriter, r *http.Request) {

	result, cancel := response.Get()
	defer cancel()
	nodeId := r.Header.Get("nodeId")
	client := tool.ClientFac(t.client, t.prefix, nodeId)

	data, err := client.ClientList(r.Context())
	if err != nil {
		result.Code = "1001"
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return

	}
	result.Data = data
	_ = result.Json(w, http.StatusOK)
}
