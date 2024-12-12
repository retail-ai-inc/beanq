package tool

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
	"sort"
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
			baseClient: nil,
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

		clusterClient.baseClient = &BaseClient{client: nodeClient}
		clusterClient.nodeId = nodeId

		return clusterClient
	}

	if clt, ok := client.(*redis.Client); ok {
		return &Client{baseClient: BaseClient{client: clt}, prefix: prefix}
	}
	return nil
}

type BaseClient struct {
	client *redis.Client
}

func (t *BaseClient) Keys(ctx context.Context, key string) ([]string, error) {

	var (
		cursor uint64
		keys   []string
	)
	for {
		ks, ncursor := t.client.Scan(ctx, cursor, key, 50).Val()
		keys = append(keys, ks...)
		if ncursor <= 0 {
			break
		}
		cursor = ncursor
	}
	return keys, nil
}

func (t *BaseClient) NodeId(ctx context.Context) string {
	return ""
}

func (t *BaseClient) Nodes(ctx context.Context) []Node {

	return nil
}

func (t *BaseClient) KeySpace(ctx context.Context) ([]map[string]any, error) {

	spaces, err := t.client.Info(ctx, "Keyspace").Result()
	if err != nil {
		return nil, err
	}
	m := parseInfos(spaces)
	slic := make([]map[string]any, 0)
	for i, v := range m {
		nmap := make(map[string]any, 0)
		vv := strings.Split(cast.ToString(v), ",")
		for _, sv := range vv {
			lv := strings.Split(sv, "=")
			nmap[lv[0]] = lv[1]
		}
		nmap["dbname"] = i
		slic = append(slic, nmap)
	}
	return slic, nil
}

func (t *BaseClient) Memory(ctx context.Context) (map[string]any, error) {

	memory, err := t.client.Info(ctx, "MEMORY").Result()
	if err != nil {
		return nil, err
	}
	m := parseInfos(memory)
	return m, nil
}

func (t *BaseClient) CommandStats(ctx context.Context) ([]map[string]any, error) {

	command, err := t.client.Info(ctx, "Commandstats").Result()
	if err != nil {
		return nil, err
	}

	if strings.Contains(command, "\r\n") {
		command = strings.ReplaceAll(command, "\r\n", "\n")
	}
	commands := strings.Split(command, "\n")

	var commandMap Commands
	for _, m := range commands[1:] {
		if !strings.Contains(m, ":") {
			continue
		}
		s := strings.Split(m, ":")
		key := strings.ReplaceAll(s[0], "cmdstat_", "")
		val := s[1]
		vals := strings.Split(val, ",")
		nmap := make(map[string]any, 0)
		nmap["command"] = key
		for _, v := range vals {
			vv := strings.Split(v, "=")
			nmap[vv[0]] = vv[1]
		}
		commandMap = append(commandMap, nmap)
	}
	sort.Sort(commandMap)
	return commandMap, nil
}

func (t *BaseClient) Persistence(ctx context.Context) (map[string]any, error) {

	persistence, err := t.client.Info(ctx, "Persistence").Result()
	if err != nil {
		return nil, err
	}
	return parseInfos(persistence), nil
}

func (t *BaseClient) Server(ctx context.Context) (map[string]any, error) {

	server, err := t.client.Info(ctx, "Server").Result()
	if err != nil {
		return nil, err
	}
	return parseInfos(server), nil
}

func (t *BaseClient) Clients(ctx context.Context) (map[string]any, error) {

	clients, err := t.client.Info(ctx, "Clients").Result()
	if err != nil {
		return nil, err
	}
	m := parseInfos(clients)
	return m, nil
}

func (t *BaseClient) Stats(ctx context.Context) (map[string]any, error) {

	stats, err := t.client.Info(ctx, "Stats").Result()
	if err != nil {
		return nil, err
	}
	return parseInfos(stats), nil
}

func (t *BaseClient) DbSize(ctx context.Context) (int64, error) {
	return t.client.DBSize(ctx).Result()
}

func (t *BaseClient) Info(ctx context.Context) (map[string]string, error) {

	infoStr, err := t.client.Info(ctx).Result()
	if err != nil {
		return nil, err
	}
	info := make(map[string]string)
	lines := strings.Split(infoStr, "\r\n")
	for _, l := range lines {
		kv := strings.Split(l, ":")
		if len(kv) == 2 {
			info[kv[0]] = kv[1]
		}
	}
	return info, nil
}

func (t *BaseClient) ClientList(ctx context.Context) ([]map[string]any, error) {

	cmd := t.client.ClientList(ctx)
	if err := cmd.Err(); err != nil {
		return nil, err
	}
	data, err := cmd.Result()
	if err != nil {
		return nil, err
	}

	arr := strings.Split(data, "\n")
	ldata := make(map[string]any, 0)
	rdata := make([]map[string]any, 0, 10)
	for _, v := range arr {
		nv := strings.Split(v, " ")
		for _, nvv := range nv {
			vals := strings.Split(nvv, "=")
			if vals[0] == "age" {
				if vals[1] == "0" {
					continue
				}
			}
			if len(vals) < 2 {
				continue
			}
			ldata[vals[0]] = vals[1]
			rdata = append(rdata, ldata)
		}
	}
	return rdata, nil
}

func (t *BaseClient) Object(ctx context.Context, queueName string) (objstr *ObjectStruct, err error) {

	obj := t.client.DebugObject(ctx, queueName)

	str, err := obj.Result()
	if err != nil {
		return nil, err
	}
	// Value at:0x7fc38fe77cc0 refcount:1 encoding:stream serializedlength:12 lru:7878503 lru_seconds_idle:3
	valueAt := "Value at"
	if strings.HasPrefix(str, valueAt) {
		str = strings.ReplaceAll(str, valueAt, "value_at")
	}

	strs := strings.Split(str, " ")

	for _, s := range strs {
		sarr := strings.Split(s, ":")
		if len(sarr) >= 2 {
			switch sarr[0] {
			case "value_at":
				objstr.ValueAt = sarr[1]
			case "refcount":
				objstr.RefCount = cast.ToInt(sarr[1])
			case "encoding":
				objstr.Encoding = sarr[1]
			case "serializedlength":
				objstr.SerizlizedLength = cast.ToInt(sarr[1])
			case "lru":
				objstr.Lru = cast.ToInt(sarr[1])
			case "lru_seconds_idle":
				objstr.LruSecondsIdle = cast.ToInt(sarr[1])
			}
		}
	}
	return
}

func (t *BaseClient) ZCard(ctx context.Context, key string) (int64, error) {
	return t.client.ZCard(ctx, key).Val(), nil
}

func (t *BaseClient) Monitor(ctx context.Context) (string, error) {
	return t.client.Do(ctx, "MONITOR").String(), nil
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
