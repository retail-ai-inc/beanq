package beanq

import (
	"context"
	"time"
)

// Publisher is the stable publish-side API exposed by Client.BQ().
type Publisher interface {
	Publish(channel, topic string, payload []byte) error
	PublishAtTime(channel, topic string, payload []byte, atTime time.Time) error
	PublishInSequence(channel, topic string, payload []byte) *SequenceCmd
	PublishInSequenceByLock(channel, topic, orderKey string, payload []byte) *SequenceCmd
}

// Consumer is the stable subscribe-side API exposed by Client.BQ().
type Consumer interface {
	Subscribe(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error)
	SubscribeToDelay(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error)
	SubscribeToSequence(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error)
	SubscribeToSequenceByLock(channel, topic string, handle IConsumeHandle) (IBaseSubscribeCmd, error)
}

// Locker describes lock operations that can be backed by Redis or another store.
type Locker interface {
	ForceUnlock(ctx context.Context, channel, topic, orderKey string) error
}

// MessageStore checks message state without exposing the backing driver.
type MessageStore interface {
	Status(ctx context.Context, channel, topic, id string, isOrder bool) (map[string]string, error)
}

// MigrationRunner runs data or schema migrations behind a small boundary.
type MigrationRunner interface {
	Migrate(ctx context.Context, data []map[string]any) error
}
