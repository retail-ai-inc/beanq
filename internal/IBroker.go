package public

import (
	"context"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
)

// IBroker main job
type (
	Stream struct {
		Data    map[string]any
		Id      string
		Channel string
		Stream  string
	}
	CallBack func(ctx context.Context, data map[string]any) error
	IBroker  interface {
		Enqueue(ctx context.Context, data map[string]any) error
		Dequeue(ctx context.Context, channel, topic string, do CallBack)
	}
	IDeadLetter interface {
		DeadLetter(ctx context.Context, channel, topic string)
	}
	IBrokerFactory interface {
		Mood(moodType btype.MoodType) IBroker
	}
)

// IProcessLog process log
type (
	IProcessLog interface {
		AddLog(ctx context.Context, data map[string]any) error
	}
	// IMigrateLog migrate redis log to other db
	// for example: to mongodb
	IMigrateLog interface {
		Migrate(ctx context.Context, data []map[string]any) error
	}
)

// IStatus check the status of the message based on the ID
type IStatus interface {
	Status(ctx context.Context, channel, topic, id string) (map[string]string, error)
}
