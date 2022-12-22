package beanq

import (
	"time"

	"beanq/helper/stringx"
	"github.com/go-redis/redis/v8"
)

type Message struct {
	Id      string
	Stream  string
	Payload []byte
}

type Task struct {
	id          string    `json:"id"`
	name        string    `json:"name"`
	queue       string    `json:"queue"`
	maxLen      int64     `json:"maxLen"`
	retry       int       `json:"retry"`
	payload     []byte    `json:"payload"`
	addTime     string    `json:"addTime"`
	executeTime time.Time `json:"executeTime"`
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

// set val functions
type TaskOpt func(task *Task)

func SetId(id string) TaskOpt {
	return func(task *Task) {
		task.id = id
	}
}
func SetName(name string) TaskOpt {
	return func(task *Task) {
		task.name = name
	}
}
func SetQueue(queue string) TaskOpt {
	return func(task *Task) {
		task.queue = queue
	}
}
func SetMaxLen(maxlen int64) TaskOpt {
	return func(task *Task) {
		task.maxLen = maxlen
	}
}
func SetRetry(retry int) TaskOpt {
	return func(task *Task) {
		task.retry = retry
	}
}
func SetPayLoad(payload []byte) TaskOpt {
	return func(task *Task) {
		task.payload = payload
	}
}
func SetAddTime(addtime string) TaskOpt {
	return func(task *Task) {
		task.addTime = addtime
	}
}
func SetExecuteTime(executeTime time.Time) TaskOpt {
	return func(task *Task) {
		task.executeTime = executeTime
	}
}

func NewTask(opt ...TaskOpt) *Task {
	task := Task{}
	for _, o := range opt {
		o(&task)
	}
	return &task
}

type DoConsumer func(*Task, *redis.Client) error
