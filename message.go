// MIT License

// Copyright The RAI Inc.
// The RAI Authors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package beanq

import (
	"strconv"
	"time"

	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/spf13/cast"
)

type (
	Message struct {
		ExecuteTime  time.Time        `json:"executeTime"`
		EndTime      time.Time        `json:"endTime"`
		BeginTime    time.Time        `json:"beginTime"`
		Response     any              `json:"response"`
		Info         bstatus.FlagInfo `json:"info"`
		Level        bstatus.LevelMsg `json:"level"`
		Topic        string           `json:"topic"`
		Channel      string           `json:"channel"`
		OrderKey     string           `json:"orderKey"`
		Payload      string           `json:"payload"`
		AddTime      string           `json:"addTime"`
		Consumer     string           `json:"consumer"`
		RunTime      string           `json:"runTime"`
		MoodType     btype.MoodType   `json:"moodType"`
		Status       bstatus.Status   `json:"status"`
		Id           string           `json:"id"`
		Retry        int              `json:"retry"`
		TimeToRun    time.Duration    `json:"timeToRun"`
		MaxLen       int64            `json:"maxLen"`
		Priority     float64          `json:"priority"`
		PendingRetry int64            `json:"pendingRetry"`
	}
)

func (m Message) MarshalBinary() (data []byte, err error) {
	return json.Marshal(m)
}

func (m Message) ToMap() map[string]any {
	data := make(map[string]any)
	data["id"] = m.Id
	data["topic"] = m.Topic
	data["channel"] = m.Channel
	data["orderKey"] = m.OrderKey
	data["maxLen"] = m.MaxLen
	data["retry"] = m.Retry
	data["priority"] = m.Priority
	data["payload"] = m.Payload
	data["addTime"] = m.AddTime
	data["executeTime"] = m.ExecuteTime
	data["moodType"] = m.MoodType
	data["timeToRun"] = m.TimeToRun
	return data
}

type MessageM map[string]any

func (data MessageM) ToMessage() *Message {
	now := time.Now()
	var msg = &Message{}
	msg.ExecuteTime = now
	for key, val := range data {
		switch key {
		case "id":
			if v, ok := val.(string); ok {
				msg.Id = v
			}
		case "topic":
			if v, ok := val.(string); ok {
				msg.Topic = v
			}
		case "channel":
			if v, ok := val.(string); ok {
				msg.Channel = v
			}
		case "maxLen":
			if v, ok := val.(int64); ok {
				msg.MaxLen = v
			}
		case "retry":
			if v, ok := val.(string); ok {
				retry, _ := strconv.Atoi(v)
				msg.Retry = retry
			}
		case "priority":
			msg.Priority = cast.ToFloat64(val)
		case "payload":
			if v, ok := val.(string); ok {
				msg.Payload = v
			}
		case "addTime":
			if v, ok := val.(string); ok {
				msg.AddTime = v
			}
		case "executeTime":
			if v, ok := val.(time.Time); ok {
				if v.IsZero() {
					msg.ExecuteTime = now
				} else {
					msg.ExecuteTime = v
				}
			}
		case "moodType":
			if v, ok := val.(string); ok {
				msg.MoodType = btype.MoodType(v)
			}
		case "timeToRun":
			if v, ok := val.(string); ok {
				dur, _ := strconv.Atoi(v)
				msg.TimeToRun = time.Duration(dur)
			}
		case "response":
			msg.Response = val
		}
	}
	return msg
}

type MessageS map[string]string

func (data MessageS) ToMessage() *Message {

	msg := Message{}
	for k, v := range data {
		if k == "id" {
			msg.Id = v
		}
		if k == "topic" {
			msg.Topic = v
		}
		if k == "channel" {
			msg.Channel = v
		}
		if k == "consumer" {
			msg.Consumer = v
		}
		if k == "retry" {
			msg.Retry = cast.ToInt(v)
		}
		if k == "pendingRetry" {
			msg.PendingRetry = cast.ToInt64(v)
		}
		if k == "priority" {
			msg.Priority = cast.ToFloat64(v)
		}
		if k == "payload" {
			msg.Payload = v
		}
		if k == "addTime" {
			msg.AddTime = v
		}
		if k == "executeTime" {
			msg.ExecuteTime = cast.ToTime(v)
		}
		if k == "moodType" {
			msg.MoodType = btype.MoodType(v)
		}
		if k == "status" {
			msg.Status = v
		}
		if k == "level" {
			msg.Level = v
		}
		if k == "info" {
			msg.Info = v
		}
		if k == "runTime" {
			msg.RunTime = v
		}
		if k == "beginTime" {
			msg.BeginTime = cast.ToTime(v)
		}
		if k == "endTime" {
			msg.EndTime = cast.ToTime(v)
		}
		if k == "response" {
			msg.Response = v
		}
	}

	return &msg
}

// If possible, more data type judgments need to be added
func messageToStruct(message any) *Message {
	msg := new(Message)
	switch xmsg := message.(type) {
	case *redis.XMessage:
		msg = MessageM(xmsg.Values).ToMessage()
	case map[string]any:
		msg = MessageM(xmsg).ToMessage()
	case map[string]string:
		msg = MessageS(xmsg).ToMessage()
	}
	return msg
}
