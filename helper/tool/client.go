package tool

import (
	"context"
)

type Client struct {
	IClient
	prefix string
	nodeId string
}

func (t *Client) NodeId(ctx context.Context) string {

	return t.nodeId
}

func (t *Client) Nodes(ctx context.Context) []Node {

	return []Node{{
		NodeId:       t.nodeId,
		Ip:           "",
		Master:       "",
		ParentNodeId: "",
		Ping:         "",
		Pong:         "",
		Flag:         "",
		Slot:         "",
		LinkState:    "",
	}}
}
