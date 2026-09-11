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

const queueMessageScanBatch int64 = 100

func NewUITool(client redis.UniversalClient, prefix string) *UITool {

	return &UITool{
		client: client,
		prefix: prefix,
	}
}

func (t *UITool) QueueMessage(ctx context.Context) error {
	var total, pending int64
	seen := make(map[string]struct{})
	var cursor uint64
	for {
		streamKeys, nextCursor, err := t.client.Scan(ctx, cursor, t.prefix+"*:stream*", queueMessageScanBatch).Result()
		if err != nil {
			return fmt.Errorf("scan queue streams: %w", err)
		}
		cursor = nextCursor
		keys := make([]string, 0, len(streamKeys))
		keyTypes := make([]*redis.StatusCmd, 0, len(streamKeys))
		groupsCommands := make([]*redis.XInfoGroupsCmd, 0, len(streamKeys))
		lengthCommands := make([]*redis.IntCmd, 0, len(streamKeys))
		pipe := t.client.Pipeline()
		for _, streamKey := range streamKeys {
			if _, exists := seen[streamKey]; exists {
				continue
			}
			seen[streamKey] = struct{}{}
			keys = append(keys, streamKey)
			keyTypes = append(keyTypes, pipe.Type(ctx, streamKey))
			groupsCommands = append(groupsCommands, pipe.XInfoGroups(ctx, streamKey))
			lengthCommands = append(lengthCommands, pipe.XLen(ctx, streamKey))
		}
		_, _ = pipe.Exec(ctx)
		for index, streamKey := range keys {
			keyType, err := keyTypes[index].Result()
			if err != nil {
				return fmt.Errorf("read queue key type for %q: %w", streamKey, err)
			}
			if keyType != "stream" {
				continue
			}
			groups, err := groupsCommands[index].Result()
			if err != nil && err != redis.Nil {
				return fmt.Errorf("read consumer groups for %q: %w", streamKey, err)
			}
			if len(groups) > 0 {
				pending += groups[0].Pending
			}
			length, err := lengthCommands[index].Result()
			if err != nil {
				return fmt.Errorf("read stream length for %q: %w", streamKey, err)
			}
			total += length
		}
		if cursor == 0 {
			break
		}
	}
	if pending < 0 {
		pending = 0
	}
	ready := max(total-pending, 0)
	now := time.Now()
	produced, _ := t.client.Get(ctx, strings.Join([]string{t.prefix, "metrics", "total", "published"}, ":")).Int64()
	consumed, _ := t.client.Get(ctx, strings.Join([]string{t.prefix, "metrics", "total", "success"}, ":")).Int64()

	// Use a named schema so dashboard clients cannot accidentally swap metric
	// positions as the payload evolves. Keep the timestamp in milliseconds for
	// unambiguous client-side date handling.
	data, err := json.Marshal(map[string]any{
		"version":      2,
		"timestamp":    now.UnixMilli(),
		"streamLength": total,
		"pending":      pending,
		"ready":        ready,
		"produced":     produced,
		"consumed":     consumed,
	})
	if err != nil {
		return fmt.Errorf("encode queue metrics: %w", err)
	}

	totalKey := strings.Join([]string{t.prefix, "dashboard_total"}, ":")
	// Avoid filling the time series with identical snapshots while the queue is
	// idle. The timestamp is intentionally excluded from the signature.
	signature := fmt.Sprintf("%d:%d:%d:%d:%d", total, pending, ready, produced, consumed)
	lastKey := strings.Join([]string{t.prefix, "dashboard_total_last"}, ":")
	last, lastErr := t.client.Get(ctx, lastKey).Result()
	if lastErr == nil && last == signature {
		return nil
	}
	if err := t.client.ZAdd(ctx, totalKey, redis.Z{
		Score:  cast.ToFloat64(now.Unix()),
		Member: data,
	}).Err(); err != nil {
		return fmt.Errorf("store queue metrics: %w", err)
	}
	if err := t.client.Set(ctx, lastKey, signature, 48*time.Hour).Err(); err != nil {
		return fmt.Errorf("store dashboard metric signature: %w", err)
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
	expiredMembers := make([]any, 0, len(keys)/2)
	for _, key := range keys {
		if err := json.NewDecoder(strings.NewReader(key)).Decode(&data); err != nil {
			continue
		}
		if v, ok := data["hostName"]; ok {
			if cast.ToString(v) == info.Hostname {
				expiredMembers = append(expiredMembers, key)
				data = make(map[string]any, 8)
				continue
			}
		}
		if v, ok := data["expiredTime"]; ok {
			if cast.ToInt64(v) < now.Unix() {
				expiredMembers = append(expiredMembers, key)
				data = make(map[string]any, 8)
				continue
			}
		}
		data = make(map[string]any, 8)
	}
	if len(expiredMembers) > 0 {
		t.client.ZRem(ctx, hostNameKey, expiredMembers...)
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
