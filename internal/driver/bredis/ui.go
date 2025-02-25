package bredis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/retail-ai-inc/beanq/v3/helper/timex"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/spf13/cast"
	"strings"
	"time"
)

type UITool struct {
	client redis.UniversalClient
	prefix string
}

func NewUITool(client redis.UniversalClient, prefix string) *UITool {

	return &UITool{
		client: client,
		prefix: prefix,
	}
}

func (t *UITool) QueueMessage(ctx context.Context) error {

	streamkeys := t.client.Keys(ctx, strings.Join([]string{t.prefix, "*", ":stream"}, "")).Val()

	var (
		total   int64
		pending int64
		ready   int64
	)

	for _, streamkey := range streamkeys {
		val := t.client.XInfoGroups(ctx, streamkey).Val()
		if len(val) > 0 {
			pending += val[0].Pending
		}

		total += t.client.XLen(ctx, streamkey).Val()
	}
	ready = total - pending

	now := time.Now()
	data := make(map[string]any, 0)
	data["time"] = now.Format(timex.TimeOnly)
	data["total"] = total
	data["pending"] = pending
	data["ready"] = ready

	bt, err := json.Marshal(data)
	if err != nil {
		return err
	}

	totalkey := strings.Join([]string{t.prefix, "dashboard_total"}, ":")

	err = t.client.Watch(ctx, func(tx *redis.Tx) error {
		_, err := tx.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
			pipeliner.ZAdd(ctx, totalkey, &redis.Z{
				Score:  cast.ToFloat64(now.Unix()),
				Member: bt,
			})
			// always keep 5 messages
			pipeliner.ZRemRangeByRank(ctx, totalkey, 0, -6)
			return nil
		})
		return err
	}, totalkey)
	return err
}

func (t *UITool) HostName(ctx context.Context) error {

	info, err := host.Info()
	if err != nil {
		return err
	}

	memory, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	cpuCount, err := cpu.Counts(false)
	if err != nil {
		return err
	}
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return err
	}

	data := make(map[string]any, 0)
	data = map[string]any{
		"cpuCount":      cpuCount,
		"cpuPercent":    cpuPercent[0],
		"memoryCount":   memory.Total,
		"memoryTotal":   fmt.Sprintf("%.2f", float64(memory.Total/(1024*1024*1024))),
		"memoryUsed":    fmt.Sprintf("%.2f", float64(memory.Used/(1024*1024))),
		"memoryPercent": memory.UsedPercent,
	}
	bt, err := json.Marshal(data)
	if err != nil {
		return err
	}
	redisVal := make(map[string]any, 0)
	redisVal[info.Hostname] = string(bt)

	return t.client.HMSet(ctx, tool.BeanqHostName, redisVal).Err()
}
