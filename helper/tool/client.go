package tool

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
	"sort"
	"strings"
)

type Client struct {
	client *redis.Client
	prefix string
}

func (t *Client) Keys(ctx context.Context, key string) ([]string, error) {

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

func (t *Client) NodeId(ctx context.Context) string {
	return ""
}

func (t *Client) Nodes(ctx context.Context) []Node {

	return nil
}

func (t *Client) KeySpace(ctx context.Context) ([]map[string]any, error) {

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

func (t *Client) Memory(ctx context.Context) (map[string]any, error) {

	memory, err := t.client.Info(ctx, "MEMORY").Result()
	if err != nil {
		return nil, err
	}
	m := parseInfos(memory)
	return m, nil
}

func (t *Client) CommandStats(ctx context.Context) ([]map[string]any, error) {

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

func (t *Client) Persistence(ctx context.Context) (map[string]any, error) {

	persistence, err := t.client.Info(ctx, "Persistence").Result()
	if err != nil {
		return nil, err
	}
	return parseInfos(persistence), nil
}

func (t *Client) Server(ctx context.Context) (map[string]any, error) {

	server, err := t.client.Info(ctx, "Server").Result()
	if err != nil {
		return nil, err
	}
	return parseInfos(server), nil
}

func (t *Client) Clients(ctx context.Context) (map[string]any, error) {

	clients, err := t.client.Info(ctx, "Clients").Result()
	if err != nil {
		return nil, err
	}
	m := parseInfos(clients)
	return m, nil
}

func (t *Client) Stats(ctx context.Context) (map[string]any, error) {

	stats, err := t.client.Info(ctx, "Stats").Result()
	if err != nil {
		return nil, err
	}
	return parseInfos(stats), nil
}

func (t *Client) DbSize(ctx context.Context) (int64, error) {
	return t.client.DBSize(ctx).Result()
}

func (t *Client) Info(ctx context.Context) (map[string]string, error) {

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

func (t *Client) ClientList(ctx context.Context) ([]map[string]any, error) {

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

func (t *Client) Object(ctx context.Context, queueName string) (objstr *ObjectStruct, err error) {

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

func (t *Client) ZCard(ctx context.Context, key string) (int64, error) {
	return t.client.ZCard(ctx, key).Val(), nil
}

func (t *Client) Monitor(ctx context.Context) (string, error) {
	return t.client.Do(ctx, "MONITOR").String(), nil
}
