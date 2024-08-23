package beanq

import (
	"context"
	"time"

	"github.com/retail-ai-inc/beanq/helper/json"
)

type (
	FlagInfo = string
	LevelMsg = string
	Status   = string

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

	StatusPrepare    Status = "prepare"
	StatusPublished  Status = "published"
	StatusPending    Status = "pending"
	StatusReceived   Status = "received"
	StatusSuccess    Status = "success"
	StatusFailed     Status = "failed"
	StatusDeadLetter Status = "dead_letter"

	ErrLevel  LevelMsg = "error"
	InfoLevel LevelMsg = "info"
)

type ILog interface {
	// Archive log
	Archive(ctx context.Context, result *Message) error
	// Obsolete ,if log has expired ,then delete it
	Obsolete(ctx context.Context, data []map[string]any) error
}

type Log struct {
	logs []ILog
	pool *asyncPool
}

func NewLog(pool *asyncPool, logs ...ILog) *Log {
	return &Log{
		logs: logs,
		pool: pool,
	}
}

func (t *Log) Archives(ctx context.Context, result Message) error {
	for _, log := range t.logs {
		nlog := log
		if err := nlog.Archive(ctx, &result); err != nil {
			t.pool.captureException(ctx, err)
		}
	}
	return nil
}

func (t *Log) Obsoletes(ctx context.Context, datas []map[string]any) error {

	for _, log := range t.logs {
		nlog := log
		go func() {
			nlog.Obsolete(ctx, datas)
		}()
	}
	return nil
}
