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
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/spf13/cast"
)

type (
	Message struct {
		Id           string        `json:"id"`
		Topic        string        `json:"topic"`
		Channel      string        `json:"channel"`
		Consumer     string        `json:"consumer"`
		MaxLen       int64         `json:"maxLen"`
		Retry        int           `json:"retry"`
		PendingRetry int64         `json:"pendingRetry"`
		Priority     float64       `json:"priority"`
		Payload      string        `json:"payload"`
		AddTime      string        `json:"addTime"`
		ExecuteTime  time.Time     `json:"executeTime"`
		TimeToRun    time.Duration `json:"timeToRun"`
		MoodType     MoodType      `json:"moodType"` // 3 types of message: `normal`, `delay`, `sequential`
		Status       Status        `json:"status"`
		Level        LevelMsg      `json:"level"`
		Info         FlagInfo      `json:"info"`
		RunTime      string        `json:"runTime"`
		BeginTime    time.Time     `json:"beginTime"`
		EndTime      time.Time     `json:"endTime"`
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
				msg.MoodType = MoodType(v)
			}
		case "timeToRun":
			if v, ok := val.(string); ok {
				dur, _ := strconv.Atoi(v)
				msg.TimeToRun = time.Duration(dur)
			}
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
			msg.MoodType = MoodType(v)
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

// Customized function
func jsonToMessage(dataStr string) (*Message, error) {

	msg := new(Message)
	reader := strings.NewReader(dataStr)
	jn := json.NewDecoder(reader)
	if err := jn.Decode(msg); err != nil {
		return nil, err
	}

	return msg, nil
}
