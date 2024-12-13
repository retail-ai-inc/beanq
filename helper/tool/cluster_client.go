package tool

import (
	"context"
	"github.com/go-redis/redis/v8"
	"strings"
)

type (
	ClusterClient struct {
		client *redis.ClusterClient
		prefix string
		nodeId string
		IClient
	}
)

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
