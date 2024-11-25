package beanq

import (
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
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
