package beanq

import (
	"time"

	"beanq/helper/json"
	"beanq/helper/stringx"
	"beanq/helper/timex"
	"beanq/internal/options"
	"github.com/go-redis/redis/v8"
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
func (t Task) Id() string {
	return t.id
}
func (t Task) Name() string {
	return t.name
}
func (t Task) Queue() string {
	return t.queue
}
func (t Task) Group() string {
	return t.group
}
func (t Task) MaxLen() int64 {
	return t.maxLen
}
func (t Task) Retry() int {
	return t.retry
}
func (t Task) Payload() string {
	return stringx.ByteToString(t.payload)
}
func (t Task) AddTime() string {
	return t.addTime
}
func (t Task) ExecuteTime() time.Time {
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

type DoConsumer func(*Task, *redis.Client) error

func ParseTask(data []byte) Task {
	task := Task{}
	jn := json.Json

	task.id = jn.Get(data, "id").ToString()
	task.name = jn.Get(data, "name").ToString()
	task.queue = jn.Get(data, "queue").ToString()
	task.group = jn.Get(data, "group").ToString()
	task.maxLen = jn.Get(data, "maxLen").ToInt64()
	task.payload = stringx.StringToByte(jn.Get(data, "payload").ToString())
	task.retry = jn.Get(data, "retry").ToInt()
	task.priority = jn.Get(data, "priority").ToFloat64()
	task.addTime = jn.Get(data, "addtime").ToString()

	executeTimeStr := jn.Get(data, "executeTime").ToString()
	task.executeTime = cast.ToTime(executeTimeStr)

	return task
}
