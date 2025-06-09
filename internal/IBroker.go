package public

import (
	"context"

	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/retail-ai-inc/beanq/v3/internal/capture"
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
		ForceUnlock(ctx context.Context, channel, topic, orderKey string) error
	}
	IDeadLetter interface {
		DeadLetter(ctx context.Context, channel, topic string)
	}
	IBrokerFactory interface {
		Mood(moodType btype.MoodType, config *capture.Config) IBroker
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
	Status(ctx context.Context, channel, topic, id string, isOrder bool) (map[string]string, error)
}

func (CallBack) Error(ctx context.Context, err error) {
	if err != nil {
		logger.New().Error(err)
	}
}
