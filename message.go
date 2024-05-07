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
		MaxLen       int64         `json:"maxLen"`
		Retry        int           `json:"retry"`
		PendingRetry int64         `json:"pendingRetry"`
		Priority     float64       `json:"priority"`
		Payload      string        `json:"payload"`
		AddTime      string        `json:"addTime"`
		ExecuteTime  time.Time     `json:"executeTime"`
		TimeToRun    time.Duration `json:"timeToRun"`
		MoodType     string        `json:"moodType"` // 3 types of message: `normal`, `delay`, `sequential`
	}
)

var defaultMessage *Message = &Message{
	Topic:     DefaultOptions.DefaultTopic,
	Channel:   DefaultOptions.DefaultChannel,
	Payload:   "",
	MaxLen:    DefaultOptions.DefaultMaxLen,
	Retry:     DefaultOptions.JobMaxRetry,
	Priority:  DefaultOptions.Priority,
	TimeToRun: DefaultOptions.TimeToRun,
	MoodType:  string(NORMAL),
}

// If possible, more data type judgments need to be added
func messageToStruct(message any) *Message {

	msg := new(Message)
	switch message.(type) {
	case *redis.XMessage:
		xmsg := message.(*redis.XMessage)
		mapToMessage(xmsg.Values, msg)
	case map[string]any:
		mapToMessage(message.(map[string]any), msg)
	}
	return msg

}

// Customized function
func mapToMessage(data map[string]any, msg *Message) {

	now := time.Now()
	msg.ExecuteTime = now
	for key, val := range data {
		switch key {
		case "id":
			if v, ok := val.(string); ok {
				msg.Id = v
			}

		case "topicName":
			if v, ok := val.(string); ok {
				msg.Topic = v
			}
		case "channelName":
			if v, ok := val.(string); ok {
				msg.Channel = v
			}
		case "maxLen":
			if v, ok := val.(int64); ok {
				msg.MaxLen = v
			}
		case "retry":
			if v, ok := val.(int); ok {
				msg.Retry = v
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
				msg.MoodType = v
			}
		}
	}
}

// Customized function
func messageToMap(message *Message) map[string]any {

	m := make(map[string]any)
	m["id"] = message.Id
	m["topicName"] = message.Topic
	m["channelName"] = message.Channel
	m["maxLen"] = message.MaxLen
	m["retry"] = message.Retry
	m["priority"] = message.Priority
	m["payload"] = message.Payload
	m["addTime"] = message.AddTime
	m["executeTime"] = message.ExecuteTime
	m["moodType"] = message.MoodType
	return m

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
