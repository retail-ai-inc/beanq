package tool

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
	"sort"
	"strings"
)

type (
	ClusterClient struct {
		client     *redis.ClusterClient
		prefix     string
		nodeClient *redis.Client
		nodeId     string
	}
)

func (t *ClusterClient) Keys(ctx context.Context, match string) ([]string, error) {

	var (
		cursor uint64
		keys   []string
	)
	for {
		ks, ncursor := t.nodeClient.Scan(ctx, cursor, match, 50).Val()
		keys = append(keys, ks...)
		if ncursor <= 0 {
			break
		}
		cursor = ncursor
	}
	return keys, nil
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

	spaces := t.nodeClient.Info(ctx, "Keyspace").Val()

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

func (t *ClusterClient) Memory(ctx context.Context) (map[string]any, error) {

	val := t.nodeClient.Info(ctx, "MEMORY").Val()
	return parseInfos(val), nil

}

func (t *ClusterClient) CommandStats(ctx context.Context) ([]map[string]any, error) {

	command := t.nodeClient.Info(ctx, "Commandstats").Val()
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

func (t *ClusterClient) Persistence(ctx context.Context) (map[string]any, error) {

	persistence := t.nodeClient.Info(ctx, "Persistence").Val()
	return parseInfos(persistence), nil

}

func (t *ClusterClient) Server(ctx context.Context) (map[string]any, error) {

	server := t.nodeClient.Info(ctx, "Server").Val()
	return parseInfos(server), nil

}

func (t *ClusterClient) Clients(ctx context.Context) (map[string]any, error) {

	clients := t.nodeClient.Info(ctx, "Clients").Val()
	m := parseInfos(clients)
	return m, nil

}

func (t *ClusterClient) Stats(ctx context.Context) (map[string]any, error) {

	stats := t.nodeClient.Info(ctx, "Stats").Val()
	return parseInfos(stats), nil

}

func (t *ClusterClient) DbSize(ctx context.Context) (int64, error) {

	return t.nodeClient.DBSize(ctx).Val(), nil

}

func (t *ClusterClient) Info(ctx context.Context) (map[string]string, error) {

	infoStr := t.nodeClient.Info(ctx).Val()

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

func (t *ClusterClient) ClientList(ctx context.Context) ([]map[string]any, error) {

	data := t.nodeClient.ClientList(ctx).Val()

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

func (t *ClusterClient) Object(ctx context.Context, queueName string) (objstr *ObjectStruct, err error) {

	str := t.nodeClient.DebugObject(ctx, queueName).Val()
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
	return objstr, nil
}

func (t *ClusterClient) ZCard(ctx context.Context, key string) (int64, error) {

	return t.nodeClient.ZCard(ctx, key).Val(), nil

}

func (t *ClusterClient) Monitor(ctx context.Context) (string, error) {

	return t.nodeClient.Do(ctx, "MONITOR").String(), nil

}
