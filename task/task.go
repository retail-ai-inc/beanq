package task

import (
	"time"

	"beanq/internal/stringx"
	"github.com/go-redis/redis/v8"
)

type Message struct {
	Id      string
	Stream  string
	Payload []byte
}

type Task struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Queue       string    `json:"queue"`
	MaxLen      int64     `json:"maxLen"`
	Retry       int       `json:"retry"`
	Payload     []byte    `json:"payload"`
	AddTime     string    `json:"addTime"`
	ExecuteTime time.Time `json:"executeTime"`
}

func NewTask(name string, payload []byte) *Task {
	return &Task{
		Name:    name,
		Payload: payload,
	}
}
func (t Task) GName() string {
	return t.Name
}

func (t Task) GPayload() string {
	return stringx.ByteToString(t.Payload)
}

func (t Task) GId() string {
	return t.Id
}
func (t Task) GAddTime() string {
	return t.AddTime
}

type FlagInfo string
type levelMsg string

const (
	SuccessInfo FlagInfo = "success"
	FailedInfo  FlagInfo = "failed"

	ErrLevel  levelMsg = "error"
	InfoLevel levelMsg = "info"
)

//
//  ConsumerResult
//  @Description:

type ConsumerResult struct {
	Level   levelMsg
	Info    FlagInfo
	Payload any

	AddTime string
	RunTime string

	Queue, Group, Consumer string
}

// need more parameters
type Result struct {
	Id   string
	Args []any
}
type Options struct {
	RedisOptions *redis.Options

	KeepJobInQueue           time.Duration
	KeepFailedJobsInHistory  time.Duration
	KeepSuccessJobsInHistory time.Duration

	MinWorkers  int
	JobMaxRetry int
	Prefix      string

	DefaultQueueName, DefaultGroup string
	DefaultMaxLen                  int64

	DefaultDelayQueueName, DefaultDelayGroup string

	RetryTime time.Duration
	WorkCount chan struct{}
}

var DefaultOptions = &Options{
	KeepJobInQueue:           7 * 1440 * time.Minute,
	KeepFailedJobsInHistory:  7 * 1440 * time.Minute,
	KeepSuccessJobsInHistory: 7 * 1440 * time.Minute,
	MinWorkers:               10,
	JobMaxRetry:              3,
	Prefix:                   "beanq",

	DefaultQueueName: "default-queue",
	DefaultGroup:     "default-group",
	DefaultMaxLen:    1000,

	DefaultDelayQueueName: "default-delay-queue",
	DefaultDelayGroup:     "default-delay-group",

	RetryTime: 800 * time.Millisecond,
	WorkCount: make(chan struct{}, 20),
}

type DoConsumer func(*Task, *redis.Client) error
