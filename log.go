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
	Status   string

	ConsumerResult struct {
		Status  Status
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
		MoodType                 MoodType
	}
)

func (c *ConsumerResult) MarshalBinary() (data []byte, err error) {
	return json.Marshal(c)
}

func (c *ConsumerResult) FillInfoByMessage(message *Message) *ConsumerResult {
	if c == nil {
		return &ConsumerResult{}
	}
	if message == nil {
		return c
	}

	c.Id = message.Id
	c.AddTime = message.AddTime
	c.Payload = message.Payload
	c.Priority = message.Priority
	c.ExecuteTime = message.ExecuteTime
	c.Topic = message.Topic
	c.Channel = message.Channel
	c.MoodType = message.MoodType
	return c
}

const (
	SuccessInfo FlagInfo = "success"
	FailedInfo  FlagInfo = "failed"

	StatusSuccess   Status = "success"
	StatusFailed    Status = "failed"
	StatusPending   Status = "pending"
	StatusExecuting Status = "executing"

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
		go func() {
			nlog.Obsolete(ctx)
		}()
	}
	return nil
}
