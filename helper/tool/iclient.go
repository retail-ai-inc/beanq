package tool

import (
	"context"
)

type (
	Key struct {
		NodeId string
	}
	Node struct {
		NodeId       string
		Ip           string
		Master       string
		ParentNodeId string
		Ping         string
		Pong         string
		Flag         string
		Slot         string
		LinkState    string
	}

	ObjectStruct struct {
		ValueAt          string
		Encoding         string
		RefCount         int
		SerizlizedLength int
		Lru              int
		LruSecondsIdle   int
	}
	// INode redis cluster node
	INode interface {
		Nodes(ctx context.Context) []Node
		NodeId(ctx context.Context) string
	}
	// IInfo redis info command
	IInfo interface {
		KeySpace(ctx context.Context) ([]map[string]any, error)
		Memory(ctx context.Context) (map[string]any, error)
		CommandStats(ctx context.Context) ([]map[string]any, error)
		Persistence(ctx context.Context) (map[string]any, error)
		Server(ctx context.Context) (map[string]any, error)
		Clients(ctx context.Context) (map[string]any, error)
		Stats(ctx context.Context) (map[string]any, error)
		Monitor(ctx context.Context) (string, error)
		DbSize(ctx context.Context) (int64, error)

		Info(ctx context.Context) (map[string]string, error)
	}
	// ICommand redis commands
	ICommand interface {
		Keys(ctx context.Context, key string) ([]string, error)
		Object(ctx context.Context, queueName string) (*ObjectStruct, error)
		ClientList(ctx context.Context) ([]map[string]any, error)
		ZCard(ctx context.Context, key string) (int64, error)
		ZRangeByScore(ctx context.Context, key string, min, max string, offset, count int64) ([]string, error)
		ZCount(ctx context.Context, key string, min, max string) int64
	}

	IClient interface {
		INode
		IInfo
		ICommand
	}
)
