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
	"time"

	"github.com/google/uuid"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/retail-ai-inc/beanq/helper/timex"
	"github.com/spf13/cast"
)

// Message values APPOINTMENT:
// Message["id"] =>   job's id
// Message["name"] => job's name
// Message["topic"] => job's topic name
// Message["channel"] => job's channel name
// Message["maxLen"] =>  upper limit `stream`
// Message["retry"] => retry count
// Message["priority"] => attribute priority;0-10;The larger the value, the earlier the execution
// Message["message"] => data message
// Message["addTime"] => The time when the message was added, the default is the current time
// Message["executeTime"] => message execute time

type values map[string]any

type Message struct {
	Values values
}

// get val functions

type iMessageValue interface {
	string | int64 | time.Time | float64 | int
}

func messageGetValue[T iMessageValue](msgValues values, key string, defaultValue T) T {
	if v, ok := msgValues[key]; ok {
		if value, ok := v.(T); ok {
			return value
		}
	}
	return defaultValue
}

func (t *Message) Id() string {
	return messageGetValue(t.Values, "id", "")
}

func (t *Message) Name() string {
	return messageGetValue(t.Values, "name", "")
}

func (t *Message) Topic() string {
	return messageGetValue(t.Values, "topic", "")
}

func (t *Message) Channel() string {
	return messageGetValue(t.Values, "channel", "")
}

func (t *Message) MaxLen() int64 {
	return messageGetValue(t.Values, "maxLen", int64(0))
}

func (t *Message) Retry() int {
	return messageGetValue(t.Values, "retry", 0)
}

func (t *Message) Priority() float64 {
	return messageGetValue(t.Values, "priority", float64(0))
}

func (t *Message) Payload() string {
	if v, ok := t.Values["message"]; ok {
		if payload, ok := v.(string); ok {
			return payload
		}
	}
	return ""
}

func (t *Message) AddTime() string {
	return messageGetValue(t.Values, "addTime", "")
}

func (t *Message) ExecuteTime() time.Time {
	return messageGetValue(t.Values, "executeTime", time.Now())
}

type MessageOpt func(msg *Message)

func SetId(id string) MessageOpt {
	return func(msg *Message) {
		if id != "" {
			msg.Values["id"] = id
		}
	}
}

func SetName(name string) MessageOpt {
	return func(msg *Message) {
		if name != "" {
			msg.Values["name"] = name
		}
	}
}

func NewMessage(message []byte, opt ...MessageOpt) *Message {
	now := time.Now()
	msg := Message{
		Values: values{
			"id":          uuid.NewString(),
			"name":        DefaultOptions.DefaultTopic,
			"topic":       DefaultOptions.DefaultTopic,
			"channel":     DefaultOptions.DefaultChannel,
			"maxLen":      DefaultOptions.DefaultMaxLen,
			"retry":       DefaultOptions.JobMaxRetry,
			"priority":    1,
			"message":     stringx.ByteToString(message),
			"addTime":     now.Format(timex.DateTime),
			"executeTime": now,
		},
	}
	for _, o := range opt {
		o(&msg)
	}
	return &msg
}

type DoConsumer func(*Message) error

func jsonToMessage(dataStr string) (*Message, error) {
	data := stringx.StringToByte(dataStr)

	jn := json.Json
	executeTimeStr := jn.Get(data, "executeTime").ToString()
	msg := Message{
		Values: values{
			"id":          jn.Get(data, "id").ToString(),
			"name":        jn.Get(data, "name").ToString(),
			"topic":       jn.Get(data, "topic").ToString(),
			"channel":     jn.Get(data, "channel").ToString(),
			"maxLen":      jn.Get(data, "maxLen").ToInt64(),
			"retry":       jn.Get(data, "retry").ToInt(),
			"priority":    jn.Get(data, "priority").ToFloat64(),
			"message":     jn.Get(data, "message").ToString(),
			"addTime":     jn.Get(data, "addTime").ToString(),
			"executeTime": cast.ToTime(executeTimeStr),
		},
	}

	return &msg, nil
}

type BqMessage struct {
	ID     string
	Values map[string]interface{}
}

func openMessageMap(msg BqMessage, streamStr string) (message string, id, stream, addTime, topic, channelName string, executeTime time.Time, retry int, maxLen int64, err error) {
	id = msg.ID
	stream = streamStr

	bt, err := json.Marshal(msg.Values)
	if err != nil {
		return "", "", "", "", "", "", time.Time{}, 0, 0, err
	}

	topic = json.Json.Get(bt, "topic").ToString()
	channelName = json.Json.Get(bt, "channel").ToString()
	maxLen = json.Json.Get(bt, "maxLen").ToInt64()
	retry = json.Json.Get(bt, "retry").ToInt()
	message = json.Json.Get(bt, "message").ToString()
	addTime = json.Json.Get(bt, "addTime").ToString()
	executeTime = cast.ToTime(json.Json.Get(bt, "executeTime").ToString())
	if executeTime.IsZero() {
		executeTime = time.Now()
	}
	return
}
