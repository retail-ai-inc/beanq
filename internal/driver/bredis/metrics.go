package bredis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// recordMetric is best-effort telemetry. Counters are bucketed by minute and
// expire automatically so instrumentation cannot affect message processing.
func recordMetric(ctx context.Context, client redis.UniversalClient, prefix, name string) {
	bucket := time.Now().UTC().Truncate(time.Minute).Unix()
	key := fmt.Sprintf("%s:metrics:%s:%d", prefix, name, bucket)
	pipe := client.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 48*time.Hour)
	pipe.Incr(ctx, fmt.Sprintf("%s:metrics:total:%s", prefix, name))
	_, _ = pipe.Exec(ctx)
}
