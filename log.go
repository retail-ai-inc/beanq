package beanq

import (
	"context"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/logger"
)

type (
	FlagInfo string
	LevelMsg string

	ConsumerResult struct {
		Id      string
		Level   LevelMsg
		Info    FlagInfo
		Payload any

		PendingRetry             int64
		Retry                    int
		Priority                 float64
		AddTime                  string
		ExpireTime               time.Time
		RunTime                  string
		BeginTime                time.Time
		EndTime                  time.Time
		ExecuteTime              time.Time
		Topic, Channel, Consumer string
		MoodType                 string
	}
)

func (c *ConsumerResult) MarshalBinary() (data []byte, err error) {
	return json.Marshal(c)
}

func (c *ConsumerResult) Initialize() *ConsumerResult {
	c.Level = InfoLevel
	c.Info = SuccessInfo
	c.RunTime = ""
	return c
}

const (
	SuccessInfo FlagInfo = "success"
	FailedInfo  FlagInfo = "failed"

	ErrLevel  LevelMsg = "error"
	InfoLevel LevelMsg = "info"
)

type ILog interface {
	// Archive log
	Archive(ctx context.Context, result *ConsumerResult) error
	// Obsolete ,if log has expired ,then delete it
	Obsolete(ctx context.Context)
}

type Log struct {
	logs []ILog
	pool *ants.Pool
}

func NewLog(pool *ants.Pool, logs ...ILog) *Log {
	return &Log{
		logs: logs,
		pool: pool,
	}
}

func (t *Log) Archives(ctx context.Context, result *ConsumerResult) error {

	for _, log := range t.logs {
		nlog := log
		if err := nlog.Archive(ctx, result); err != nil {
			logger.New().Error(err)
		}
	}
	return nil
}

func (t *Log) Obsoletes(ctx context.Context) error {

	for _, log := range t.logs {
		nlog := log
		nlog.Obsolete(ctx)
	}
	return nil
}
