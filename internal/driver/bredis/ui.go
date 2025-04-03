package bredis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/retail-ai-inc/beanq/v3/helper/timex"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/spf13/cast"
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

	timer := timex.TimerPool.Get(5 * time.Second)
	defer timer.Stop()

	var (
		total   int64
		pending int64
		ready   int64
	)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:

		}
		timer.Reset(5 * time.Second)
		total, pending, ready = 0, 0, 0

		// total data from all streams
		streamkeys := t.client.Keys(ctx, strings.Join([]string{t.prefix, "*", ":stream"}, "")).Val()

		for _, streamkey := range streamkeys {
			val := t.client.XInfoGroups(ctx, streamkey).Val()
			if len(val) > 0 {
				pending += val[0].Pending
			}
			total += t.client.XLen(ctx, streamkey).Val()
		}
		if pending < 0 {
			pending = 0
		}
		if total < 0 {
			total = 0
		}
		ready = total - pending

		now := time.Now()
		data := make(map[string]any, 0)
		data["time"] = now.Format(time.DateTime)
		data["total"] = total
		data["pending"] = pending
		data["ready"] = ready

		bt, err := json.Marshal(data)
		if err != nil {
			logger.New().Error(err)
			continue
		}

		totalkey := strings.Join([]string{t.prefix, "dashboard_total"}, ":")

		if err := t.client.ZAdd(ctx, totalkey, &redis.Z{
			Score:  cast.ToFloat64(now.Unix()),
			Member: bt,
		}).Err(); err != nil {
			logger.New().Error(err)
		}
		before := now.Add(-48 * time.Hour).Unix()
		if err := t.client.ZRemRangeByScore(ctx, totalkey, "0", cast.ToString(before)).Err(); err != nil {
			logger.New().Error(err)
		}
	}
}

func (t *UITool) HostName(ctx context.Context) error {

	now := time.Now()

	info, err := host.Info()
	if err != nil {
		return err
	}

	hostNameKey := strings.Join([]string{t.prefix, tool.BeanqHostName}, ":")

	keys, _, err := t.client.ZScan(ctx, hostNameKey, 0, fmt.Sprintf("*%s*", info.Hostname), 10).Result()
	if err != nil {
		return err
	}
	data := make(map[string]any, 8)

	for _, key := range keys {
		if err := json.NewDecoder(strings.NewReader(key)).Decode(&data); err != nil {
			continue
		}
		if v, ok := data["hostName"]; ok {
			if cast.ToString(v) == info.Hostname {
				t.client.ZRem(ctx, hostNameKey, key)
				data = nil
				continue
			}
		}
		if v, ok := data["expiredTime"]; ok {
			if cast.ToInt64(v) < now.Unix() {
				t.client.ZRem(ctx, hostNameKey, key)
				data = nil
				continue
			}
		}
		data = nil
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

	data["hostName"] = info.Hostname
	data["cpuCount"] = cpuCount
	data["cpuPercent"] = fmt.Sprintf("%.2f", cpuPercent[0])
	data["memoryCount"] = memory.Total
	data["memoryTotal"] = fmt.Sprintf("%.2f", float64(memory.Total/(1024*1024*1024)))
	data["memoryUsed"] = fmt.Sprintf("%.2f", float64(memory.Used/(1024*1024)))
	data["memoryPercent"] = fmt.Sprintf("%.2f", memory.UsedPercent)
	data["expiredTime"] = now.Add(50 * time.Second).Unix()

	bt, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := t.client.ZAdd(ctx, hostNameKey, &redis.Z{
		Score:  cast.ToFloat64(now.Unix()),
		Member: bt,
	}).Err(); err != nil {
		return err
	}

	return nil
}
