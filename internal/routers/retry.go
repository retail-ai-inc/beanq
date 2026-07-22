package routers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/bredis"
)

var errUnsupportedRetryType = errors.New("queue type does not support retry")

type retryPublisherFactory func(string) publishQueue

func defaultRetryPublisherFactory(client redis.UniversalClient, prefix string) retryPublisherFactory {
	return func(queueType string) publishQueue {
		switch queueType {
		case string(btype.DELAY):
			return bredis.NewSchedule(client, prefix, 100, 10, 20*time.Minute, nil)
		case string(btype.NORMAL):
			return bredis.NewNormal(client, prefix, 2000, 100, 10, 20*time.Minute, nil)
		default:
			return nil
		}
	}
}

func publishRetry(ctx context.Context, data map[string]any, factory retryPublisherFactory) error {
	queueType, ok := data["moodType"].(string)
	if !ok || queueType == "" {
		return errors.New("moodType must be a string")
	}
	publisher := factory(queueType)
	if publisher == nil {
		return fmt.Errorf("%w: %s", errUnsupportedRetryType, queueType)
	}
	return publisher.Publish(ctx, data)
}
