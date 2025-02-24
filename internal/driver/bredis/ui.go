package bredis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/retail-ai-inc/beanq/v3/helper/timex"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/spf13/cast"
	"os"
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
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	return t.client.SAdd(ctx, tool.BeanqHostName, hostname).Err()
}
