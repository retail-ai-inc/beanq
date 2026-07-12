package beanq

import (
	"context"

	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
)

type queueGateway interface {
	Enqueue(ctx context.Context, data map[string]any) error
	Dequeue(ctx context.Context, channel, topic string, do public.CallbackWithRetry)
}

type locker interface {
	ForceUnlock(ctx context.Context, channel, topic, orderKey string) error
}

type statusReader interface {
	Status(ctx context.Context, channel, topic, id string, isOrder bool) (map[string]string, error)
}

type brokerStrategy func(moodType btype.MoodType, config *capture.Config) queueGateway
