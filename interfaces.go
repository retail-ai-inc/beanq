package beanq

import (
	"context"
	"time"

	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
)

// Broker is the transport contract implemented by each queue backend.
type Broker interface {
	Enqueue(ctx context.Context, data map[string]any) error
	Consume(ctx context.Context, moodType btype.MoodType, channel, topic string, callback public.CallbackWithRetry) error
	WaitingAck(ctx context.Context, channel, topic, id string) (map[string]string, error)
}

// Publisher is the stable publish-side API exposed by Client.BQ().
type Publisher interface {
	Publish(channel, topic string, payload []byte) error
	PublishAtTime(channel, topic string, payload []byte, atTime time.Time) error
	PublishSequence(channel, topic, orderKey string, payload []byte) *SequenceCmd
}

// Consumer is the stable subscribe-side API exposed by Client.BQ().
type Consumer interface {
	Subscribe(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error)
	SubscribeToDelay(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error)
	SubscribeSequence(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error)
}

// MessageStore checks message state without exposing the backing driver.
type MessageStore interface {
	Status(ctx context.Context, channel, topic, id string) (map[string]string, error)
}

// MigrationRunner runs data or schema migrations behind a small boundary.
type MigrationRunner interface {
	Migrate(ctx context.Context, data []map[string]any) error
}

type adminReporter interface {
	HostName(ctx context.Context) error
	QueueMessage(ctx context.Context) error
}

type driverProvider interface {
	Driver() any
}
