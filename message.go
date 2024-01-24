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

	"github.com/google/uuid"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/retail-ai-inc/beanq/helper/timex"
)

type (
	values  map[string]any
	Message struct {
		ID     string
		Values map[string]interface{}
	}
	// get val functions
	iMessageValue interface {
		string | int64 | time.Time | float64 | int
	}
)

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

func NewMessage(message []byte) *Message {
	now := time.Now()
	id := uuid.NewString()
	msg := Message{
		ID: id,
		Values: values{
			"id":          id,                            // job's id
			"name":        DefaultOptions.DefaultTopic,   // job's name
			"topic":       DefaultOptions.DefaultTopic,   // job's topic name
			"channel":     DefaultOptions.DefaultChannel, // job's channel name
			"maxLen":      DefaultOptions.DefaultMaxLen,  // upper limit `stream`
			"retry":       DefaultOptions.JobMaxRetry,    // retry count
			"priority":    1,                             // attribute priority;0-10;The larger the value, the earlier the execution
			"message":     stringx.ByteToString(message), // data message
			"addTime":     now.Format(timex.DateTime),    // The time when the message was added, the default is the current time
			"executeTime": now,                           // message execute time
		},
	}

	return &msg
}

type DoConsumer func(*Message) error

func jsonToMessage(dataStr string) (*Message, error) {

	// var msg Message
	msg := Message{
		ID:     "",
		Values: make(map[string]any),
	}
	mmsg := make(map[string]any)

	reader := strings.NewReader(dataStr)
	jn := json.NewDecoder(reader)
	if err := jn.Decode(&mmsg); err != nil {
		return nil, err
	}
	if v, ok := mmsg["id"]; ok {
		msg.ID = v.(string)
	}

	msg.Values = mmsg
	return &msg, nil
}
