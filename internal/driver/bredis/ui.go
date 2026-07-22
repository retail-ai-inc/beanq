package bredis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
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
	streamKeys, err := t.client.Keys(ctx, "*"+t.prefix+"*:stream*").Result()
	if err != nil {
		return fmt.Errorf("list queue streams: %w", err)
	}

	var total, pending int64
	for _, streamKey := range streamKeys {
		keyType, err := t.client.Type(ctx, streamKey).Result()
		if err != nil {
			return fmt.Errorf("read queue key type for %q: %w", streamKey, err)
		}
		if keyType != "stream" {
			continue
		}
		groups, err := t.client.XInfoGroups(ctx, streamKey).Result()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("read consumer groups for %q: %w", streamKey, err)
		}
		if len(groups) > 0 {
			pending += groups[0].Pending
		}

		length, err := t.client.XLen(ctx, streamKey).Result()
		if err != nil {
			return fmt.Errorf("read stream length for %q: %w", streamKey, err)
		}
		total += length
	}
	if pending < 0 {
		pending = 0
	}
	ready := max(total-pending, 0)
	now := time.Now()

	data, err := json.Marshal([]any{total, pending, ready, now.Format(time.DateTime)})
	if err != nil {
		return fmt.Errorf("encode queue metrics: %w", err)
	}

	totalKey := strings.Join([]string{t.prefix, "dashboard_total"}, ":")
	if err := t.client.ZAdd(ctx, totalKey, redis.Z{
		Score:  cast.ToFloat64(now.Unix()),
		Member: data,
	}).Err(); err != nil {
		return fmt.Errorf("store queue metrics: %w", err)
	}

	before := now.Add(-48 * time.Hour).Unix()
	if err := t.client.ZRemRangeByScore(ctx, totalKey, "0", cast.ToString(before)).Err(); err != nil {
		return fmt.Errorf("prune queue metrics: %w", err)
	}
	return nil
}

func (t *UITool) HostName(ctx context.Context) error {

	now := time.Now()

	info, err := host.Info()
	if err != nil {
		return err
	}

	hostNameKey := strings.Join([]string{t.prefix, tool.BeanqHostName}, ":")
	data := make(map[string]any, 8)
	keys, _, err := t.client.ZScan(ctx, hostNameKey, 0, "*", 20).Result()
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := json.NewDecoder(strings.NewReader(key)).Decode(&data); err != nil {
			continue
		}
		if v, ok := data["hostName"]; ok {
			if cast.ToString(v) == info.Hostname {
				t.client.ZRem(ctx, hostNameKey, key)
				data = make(map[string]any, 8)
				continue
			}
		}
		if v, ok := data["expiredTime"]; ok {
			if cast.ToInt64(v) < now.Unix() {
				t.client.ZRem(ctx, hostNameKey, key)
				data = make(map[string]any, 8)
				continue
			}
		}
		data = make(map[string]any, 8)
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

	if err := t.client.ZAdd(ctx, hostNameKey, redis.Z{
		Score:  cast.ToFloat64(now.Unix()),
		Member: bt,
	}).Err(); err != nil {
		return err
	}

	return nil
}
