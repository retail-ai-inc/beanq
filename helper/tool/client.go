package tool

import (
	"context"
)

type Client struct {
	baseClient BaseClient
	prefix     string
}

func (t *Client) Keys(ctx context.Context, key string) ([]string, error) {

	return t.baseClient.Keys(ctx, key)

}

func (t *Client) NodeId(ctx context.Context) string {
	return ""
}

func (t *Client) Nodes(ctx context.Context) []Node {

	return nil
}

func (t *Client) KeySpace(ctx context.Context) ([]map[string]any, error) {

	return t.baseClient.KeySpace(ctx)
}

func (t *Client) Memory(ctx context.Context) (map[string]any, error) {

	return t.baseClient.Memory(ctx)

}

func (t *Client) CommandStats(ctx context.Context) ([]map[string]any, error) {

	return t.baseClient.CommandStats(ctx)
}

func (t *Client) Persistence(ctx context.Context) (map[string]any, error) {

	return t.baseClient.Persistence(ctx)

}

func (t *Client) Server(ctx context.Context) (map[string]any, error) {

	return t.baseClient.Server(ctx)

}

func (t *Client) Clients(ctx context.Context) (map[string]any, error) {

	return t.baseClient.Clients(ctx)

}

func (t *Client) Stats(ctx context.Context) (map[string]any, error) {

	return t.baseClient.Stats(ctx)

}

func (t *Client) DbSize(ctx context.Context) (int64, error) {

	return t.baseClient.DbSize(ctx)

}

func (t *Client) Info(ctx context.Context) (map[string]string, error) {

	return t.baseClient.Info(ctx)

}

func (t *Client) ClientList(ctx context.Context) ([]map[string]any, error) {

	return t.baseClient.ClientList(ctx)

}

func (t *Client) Object(ctx context.Context, queueName string) (objstr *ObjectStruct, err error) {

	return t.baseClient.Object(ctx, queueName)

}

func (t *Client) ZCard(ctx context.Context, key string) (int64, error) {

	return t.baseClient.ZCard(ctx, key)

}

func (t *Client) Monitor(ctx context.Context) (string, error) {
	return t.baseClient.Monitor(ctx)
}
