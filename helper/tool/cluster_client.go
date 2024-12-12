package tool

import (
	"context"
	"github.com/go-redis/redis/v8"
	"strings"
)

type (
	ClusterClient struct {
		client     *redis.ClusterClient
		prefix     string
		baseClient *BaseClient
		nodeId     string
	}
)

func (t *ClusterClient) Keys(ctx context.Context, match string) ([]string, error) {

	return t.baseClient.Keys(ctx, match)
}

func (t *ClusterClient) NodeId(ctx context.Context) string {
	return t.nodeId
}

func (t *ClusterClient) Nodes(ctx context.Context) []Node {

	node := t.client.ClusterNodes(ctx).Val()
	nodes := strings.Split(node, "\n")
	result := make([]Node, 0, len(nodes))

	var nd Node
	var splt []string

	for _, s := range nodes {
		nd = Node{}
		splt = strings.Split(s, " ")
		if len(splt) < 3 {
			continue
		}

		nd.NodeId = splt[0]
		nd.Ip = splt[1]
		nd.Master = splt[2]

		splt = append([]string(nil), splt[3:]...)

		for i, s2 := range splt {
			if i == 0 {
				if nd.Master == "slave" {
					nd.ParentNodeId = s2
				}
			}
			if i == 1 {
				nd.Ping = s2
			}
			if i == 2 {
				nd.Pong = s2
			}
			if i == 3 {
				nd.Flag = s2
			}
			if i == 4 {
				nd.LinkState = s2
			}
			if i == 5 {
				nd.Slot = s2
			}
		}

		result = append(result, nd)
	}
	return result
}

func (t *ClusterClient) KeySpace(ctx context.Context) ([]map[string]any, error) {

	return t.baseClient.KeySpace(ctx)

}

func (t *ClusterClient) Memory(ctx context.Context) (map[string]any, error) {

	return t.baseClient.Memory(ctx)

}

func (t *ClusterClient) CommandStats(ctx context.Context) ([]map[string]any, error) {

	return t.baseClient.CommandStats(ctx)
}

func (t *ClusterClient) Persistence(ctx context.Context) (map[string]any, error) {

	return t.baseClient.Persistence(ctx)

}

func (t *ClusterClient) Server(ctx context.Context) (map[string]any, error) {

	return t.baseClient.Server(ctx)

}

func (t *ClusterClient) Clients(ctx context.Context) (map[string]any, error) {

	return t.baseClient.Clients(ctx)

}

func (t *ClusterClient) Stats(ctx context.Context) (map[string]any, error) {

	return t.baseClient.Stats(ctx)

}

func (t *ClusterClient) DbSize(ctx context.Context) (int64, error) {

	return t.baseClient.DbSize(ctx)
}

func (t *ClusterClient) Info(ctx context.Context) (map[string]string, error) {

	return t.baseClient.Info(ctx)

}

func (t *ClusterClient) ClientList(ctx context.Context) ([]map[string]any, error) {

	return t.baseClient.ClientList(ctx)

}

func (t *ClusterClient) Object(ctx context.Context, queueName string) (objstr *ObjectStruct, err error) {

	return t.baseClient.Object(ctx, queueName)

}

func (t *ClusterClient) ZCard(ctx context.Context, key string) (int64, error) {

	return t.baseClient.ZCard(ctx, key)

}

func (t *ClusterClient) Monitor(ctx context.Context) (string, error) {
	return t.baseClient.Monitor(ctx)
}
