package beanq

import (
	"context"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"time"

	"github.com/retail-ai-inc/beanq/v3/helper/json"
)

type (
	FlagInfo = string
	LevelMsg = string
	Status   = string

	ConsumerResult struct {
		ExpireTime   time.Time
		ExecuteTime  time.Time
		EndTime      time.Time
		BeginTime    time.Time
		Payload      any
		Info         bstatus.FlagInfo
		AddTime      string
		Status       bstatus.Status
		RunTime      string
		Level        bstatus.LevelMsg
		Id           string
		Topic        string
		Channel      string
		Consumer     string
		MoodType     btype.MoodType
		Retry        int
		Priority     float64
		PendingRetry int64
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
	Archive(ctx context.Context, result *Message, isSequential bool) error
	// Obsolete ,if log has expired ,then delete it
	Obsolete(ctx context.Context, data []map[string]any) error
}

type Log struct {
	pool *asyncPool
	logs []ILog
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
		if err := nlog.Archive(ctx, &result, false); err != nil {
			t.pool.captureException(ctx, err)
		}
	}
	return nil
}

func (t *Log) Obsoletes(ctx context.Context, datas []map[string]any) error {

	for _, log := range t.logs {
		nlog := log
		go func() {
			if err := nlog.Obsolete(ctx, datas); err != nil {
				logger.New().Error(err)
			}
		}()
	}
	return nil
}
