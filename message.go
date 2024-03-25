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
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/retail-ai-inc/beanq/helper/json"
	"github.com/retail-ai-inc/beanq/helper/stringx"
	"github.com/retail-ai-inc/beanq/helper/timex"
)

type (
	Message struct {
		Id           string    `json:"id"`
		TopicName    string    `json:"topicName"`
		ChannelName  string    `json:"channelName"`
		MaxLen       int64     `json:"maxLen"`
		Retry        int       `json:"retry"`
		PendingRetry int64     `json:"pendingRetry"`
		Priority     float64   `json:"priority"`
		Payload      string    `json:"payload"`
		AddTime      string    `json:"addTime"`
		ExecuteTime  time.Time `json:"executeTime"`
		MsgType      string    `json:"msgType"`
	}
)

func NewMessage(message []byte) *Message {
	now := time.Now()
	id := uuid.NewString()
	return &Message{
		Id:          id,
		TopicName:   DefaultOptions.DefaultTopic,
		ChannelName: DefaultOptions.DefaultChannel,
		MaxLen:      DefaultOptions.DefaultMaxLen,
		Retry:       DefaultOptions.JobMaxRetry,
		Priority:    0,
		Payload:     stringx.ByteToString(message),
		AddTime:     now.Format(timex.DateTime),
		ExecuteTime: now,
		MsgType:     "normal",
	}
}

type DoConsumer func(ctx context.Context, msg *Message) error

func xMessageToStruct(message *redis.XMessage) *Message {
	msg := new(Message)
	msg.Id = message.ID
	values := message.Values

	for key, val := range values {
		switch key {
		case "topicName":
			if v, ok := val.(string); ok {
				msg.TopicName = v
			}
		case "channelName":
			if v, ok := val.(string); ok {
				msg.ChannelName = v
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
			if v, ok := val.(float64); ok {
				msg.Priority = v
			}
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
				msg.ExecuteTime = v
			}
		case "msgType":
			if v, ok := val.(string); ok {
				msg.MsgType = v
			}
		}
	}
	return msg
}

func messageToMap(message *Message) (map[string]any, error) {
	m := make(map[string]any)
	b, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	d := json.NewDecoder(bytes.NewBuffer(b))
	if err := d.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func jsonToMessage(dataStr string) (*Message, error) {

	msg := new(Message)
	reader := strings.NewReader(dataStr)
	jn := json.NewDecoder(reader)
	if err := jn.Decode(msg); err != nil {
		return nil, err
	}

	return msg, nil
}
