package tool

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
	"strings"
	"sync"
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

	IClient interface {
		Nodes(ctx context.Context) []Node
		NodeId(ctx context.Context) string

		Keys(ctx context.Context, key string) ([]string, error)

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
		Object(ctx context.Context, queueName string) (*ObjectStruct, error)

		ClientList(ctx context.Context) ([]map[string]any, error)

		ZCard(ctx context.Context, key string) (int64, error)
	}
)

func ClientFac(client redis.UniversalClient, prefix, nodeId string) IClient {

	if clt, ok := client.(*redis.ClusterClient); ok {

		var mux sync.Mutex
		var nodeClient *redis.Client

		clusterClient := &ClusterClient{
			client:     clt,
			prefix:     prefix,
			nodeClient: nil,
		}
		if nodeId == "" {
			nodes := clusterClient.Nodes(context.Background())

			for _, node := range nodes {
				if strings.Contains(node.Master, "master") {
					nodeId = node.NodeId
				}
			}
		}

		_ = clt.ForEachShard(context.Background(), func(ctx2 context.Context, client *redis.Client) error {
			mux.Lock()
			id := client.Do(ctx2, "CLUSTER", "MYID").Val()

			if cast.ToString(id) == nodeId {
				nodeClient = client
			}
			mux.Unlock()
			return nil
		})

		clusterClient.nodeClient = nodeClient
		clusterClient.nodeId = nodeId
		
		return clusterClient
	}

	if clt, ok := client.(*redis.Client); ok {
		return &Client{client: clt, prefix: prefix}
	}
	return nil
}

func parseInfos(data string) map[string]any {
	if strings.Contains(data, "\r\n") {
		data = strings.ReplaceAll(data, "\r\n", "\n")
	}
	dataM := make(map[string]any, 0)
	arr := strings.Split(data, "\n")
	for _, m := range arr[1:] {
		if !strings.Contains(m, ":") {
			continue
		}
		s := strings.Split(m, ":")
		dataM[s[0]] = s[1]
	}
	return dataM
}
