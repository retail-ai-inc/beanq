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
	"sync"
	"time"

	"beanq/helper/json"
	"beanq/helper/stringx"
	"beanq/helper/timex"
	"beanq/internal/options"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

// task values APPOINTMENT:
// task["id"] =>   job's id
// task["name"] => job's name
// task["queue"] => job's queue name
// task["group"] => job's group name
// task["maxLen"] =>  upper limit `stream`
// task["retry"] => retry count
// task["priority"] => attribute priority;0-10;The larger the value, the earlier the execution
// task["payload"] => data payload
// task["addTime"] => The time when the task was added, the default is the current time
// task["executeTime"] => task execute time

type values map[string]any

type Task struct {
	Values values
	rw     *sync.RWMutex
}

// get val functions

type iTaskValue interface {
	string | int64 | time.Time | float64 | int
}

func taskGetValue[T iTaskValue](taskValues values, key string, defaultValue T) T {
	if v, ok := taskValues[key]; ok {
		if value, ok := v.(T); ok {
			return value
		}
	}
	return defaultValue
}

func (t *Task) Id() string {
	return taskGetValue(t.Values, "id", "")
}

func (t *Task) Name() string {
	return taskGetValue(t.Values, "name", "")
}

func (t *Task) Queue() string {
	return taskGetValue(t.Values, "queue", "")
}

func (t *Task) Group() string {
	return taskGetValue(t.Values, "group", "")
}

func (t *Task) MaxLen() int64 {
	return taskGetValue(t.Values, "maxLen", int64(0))
}

func (t *Task) Retry() int {
	return taskGetValue(t.Values, "retry", 0)
}

func (t *Task) Priority() float64 {
	return taskGetValue(t.Values, "priority", float64(0))
}

func (t *Task) Payload() string {
	if v, ok := t.Values["payload"]; ok {
		if payload, ok := v.([]byte); ok {
			return string(payload)
		}
	}
	return ""
}

func (t *Task) AddTime() string {
	return taskGetValue(t.Values, "addTime", "")
}

func (t *Task) ExecuteTime() time.Time {
	return taskGetValue(t.Values, "executeTime", time.Now())
}

type TaskOpt func(task *Task)

func SetId(id string) TaskOpt {
	return func(task *Task) {
		if id != "" {
			task.Values["id"] = id
		}
	}
}

func SetName(name string) TaskOpt {
	return func(task *Task) {
		if name != "" {
			task.Values["name"] = name
		}
	}
}

func NewTask(payload []byte, opt ...TaskOpt) *Task {
	now := time.Now()
	task := Task{
		Values: values{
			"id":          uuid.NewString(),
			"name":        options.DefaultOptions.DefaultQueueName,
			"queue":       options.DefaultOptions.DefaultQueueName,
			"group":       options.DefaultOptions.DefaultGroup,
			"maxLen":      options.DefaultOptions.DefaultMaxLen,
			"retry":       options.DefaultOptions.JobMaxRetry,
			"priority":    1,
			"payload":     stringx.ByteToString(payload),
			"addTime":     now.Format(timex.DateTime),
			"executeTime": now,
		},
		rw: new(sync.RWMutex),
	}
	for _, o := range opt {
		o(&task)
	}
	return &task
}

type DoConsumer func(*Task) error

func jsonToTask(data []byte) *Task {
	jn := json.Json
	executeTimeStr := jn.Get(data, "executeTime").ToString()
	task := Task{
		Values: values{
			"id":          jn.Get(data, "id").ToString(),
			"name":        jn.Get(data, "name").ToString(),
			"queue":       jn.Get(data, "queue").ToString(),
			"group":       jn.Get(data, "group").ToString(),
			"maxLen":      jn.Get(data, "maxLen").ToInt64(),
			"retry":       jn.Get(data, "retry").ToInt(),
			"priority":    jn.Get(data, "priority").ToFloat64(),
			"payload":     jn.Get(data, "payload").ToString(),
			"addTime":     jn.Get(data, "addTime").ToString(),
			"executeTime": cast.ToTime(executeTimeStr),
		},
		rw: new(sync.RWMutex),
	}

	return &task
}

type BqMessage struct {
	ID     string
	Values map[string]interface{}
}

func openTaskMap(msg BqMessage, streamStr string) (payload []byte, id, stream, addTime, queue, group string, executeTime time.Time, retry int, maxLen int64) {
	id = msg.ID
	stream = streamStr

	if queueVal, ok := msg.Values["queue"]; ok {
		if v, ok := queueVal.(string); ok {
			queue = v
		}
	}

	if groupVal, ok := msg.Values["group"]; ok {
		if v, ok := groupVal.(string); ok {
			group = v
		}
	}

	if maxLenV, ok := msg.Values["maxLen"]; ok {
		if v, ok := maxLenV.(string); ok {
			maxLen = cast.ToInt64(v)
		}
	}

	if retryVal, ok := msg.Values["retry"]; ok {
		if v, ok := retryVal.(string); ok {
			retry = cast.ToInt(v)
		}
	}

	if payloadVal, ok := msg.Values["payload"]; ok {
		if payloadV, ok := payloadVal.(string); ok {
			payload = stringx.StringToByte(payloadV)
		}
	}

	if addtimeV, ok := msg.Values["addTime"]; ok {
		if addtimeStr, ok := addtimeV.(string); ok {
			addTime = addtimeStr
		}
	}

	if executeTVal, ok := msg.Values["executeTime"]; ok {
		if executeTm, ok := executeTVal.(string); ok {
			executeTime = cast.ToTime(executeTm)
		}
	}

	return
}
