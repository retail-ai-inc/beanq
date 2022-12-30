package beanq

import (
	"time"

	"beanq/helper/json"
	"beanq/helper/stringx"
	"beanq/helper/timex"
	"beanq/internal/options"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

type Message struct {
	Id      string
	Stream  string
	Payload []byte
}

type Task struct {
	id    string
	name  string
	queue string

	group    string
	maxLen   int64
	retry    int
	priority float64

	payload     []byte
	addTime     string
	executeTime time.Time
}

// get val functions
func (t *Task) Id() string {
	return t.id
}
func (t *Task) Name() string {
	return t.name
}
func (t *Task) Queue() string {
	return t.queue
}
func (t *Task) Group() string {
	return t.group
}
func (t *Task) MaxLen() int64 {
	return t.maxLen
}
func (t *Task) Retry() int {
	return t.retry
}
func (t *Task) Priority() float64 {
	return t.priority
}
func (t *Task) Payload() string {
	return stringx.ByteToString(t.payload)
}
func (t *Task) AddTime() string {
	return t.addTime
}
func (t *Task) ExecuteTime() time.Time {
	return t.executeTime
}

type TaskOpt func(task *Task)

func SetId(id string) TaskOpt {
	return func(task *Task) {
		if id != "" {
			task.id = id
		}
	}
}
func SetName(name string) TaskOpt {
	return func(task *Task) {
		if name != "" {
			task.name = name
		}
	}
}

func NewTask(payload []byte, opt ...TaskOpt) *Task {
	now := time.Now()
	task := Task{
		id:          uuid.NewString(),
		name:        options.DefaultOptions.DefaultQueueName,
		queue:       options.DefaultOptions.DefaultQueueName,
		group:       options.DefaultOptions.DefaultGroup,
		maxLen:      options.DefaultOptions.DefaultMaxLen,
		retry:       options.DefaultOptions.JobMaxRetry,
		priority:    1,
		payload:     payload,
		addTime:     now.Format(timex.DateTime),
		executeTime: now,
	}

	for _, o := range opt {
		o(&task)
	}
	return &task
}

type DoConsumer func(*Task) error

/*
* jsonToTask
*  @Description:
* @param data
* @return Task
 */
func jsonToTask(data []byte) Task {

	jn := json.Json
	executeTimeStr := jn.Get(data, "executeTime").ToString()

	task := Task{
		id:          jn.Get(data, "id").ToString(),
		name:        jn.Get(data, "name").ToString(),
		queue:       jn.Get(data, "queue").ToString(),
		group:       jn.Get(data, "group").ToString(),
		maxLen:      jn.Get(data, "maxLen").ToInt64(),
		retry:       jn.Get(data, "retry").ToInt(),
		priority:    jn.Get(data, "priority").ToFloat64(),
		payload:     stringx.StringToByte(jn.Get(data, "payload").ToString()),
		addTime:     jn.Get(data, "addtime").ToString(),
		executeTime: cast.ToTime(executeTimeStr),
	}
	return task
}

/*
* makeTaskMap
*  @Description:
* @param id
* @param queue
* @param name
* @param payload
* @param group
* @param retry
* @param priority
* @param maxLen
* @param executeTime
* @return map[string]any
 */
func makeTaskMap(id, queue, name, payload, group string, retry int, priority float64, maxLen int64, executeTime time.Time) map[string]any {
	now := time.Now()
	values := make(map[string]any)
	values["id"] = id
	values["queue"] = queue
	values["name"] = name
	values["payload"] = payload
	values["addtime"] = now.Format(timex.DateTime)
	values["retry"] = retry
	values["maxLen"] = maxLen
	values["group"] = group
	values["priority"] = priority

	if executeTime.IsZero() {
		executeTime = now
	}
	values["executeTime"] = executeTime
	return values
}

type BqMessage struct {
	ID     string
	Values map[string]interface{}
}

/*
* openTaskMap
*  @Description:
* @param msg
* @param streamStr
* @return payload
* @return id
* @return stream
* @return addTime
* @return queue
* @return group
* @return executeTime
* @return retry
* @return maxLen
 */
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

	if addtimeV, ok := msg.Values["addtime"]; ok {
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
