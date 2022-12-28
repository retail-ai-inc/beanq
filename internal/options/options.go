package options

import (
	"time"

	"github.com/go-redis/redis/v8"
)

type OptionType int

const (
	MaxRetryOpt OptionType = iota + 1
	PriorityOpt
	QueueOpt
	GroupOpt
	MaxLenOpt
	ExecuteTimeOpt
	IdleTime
)

type Option struct {
	Priority    float64
	Retry       int
	Queue       string
	Group       string
	MaxLen      int64
	ExecuteTime time.Time
}

type OptionI interface {
	String() string
	OptType() OptionType
	Value() any
}

type (
	priorityOption float64
	retryOption    int
	queueOption    string
	groupOption    string
	maxLenOption   int64
	executeTime    time.Time
)

/*
* Queue
*  @Description:
* @param name
* @return Option
 */
func Queue(name string) OptionI {
	return queueOption(name)
}
func (queue queueOption) String() string {
	return "queueOption"
}
func (queue queueOption) OptType() OptionType {
	return QueueOpt
}
func (queue queueOption) Value() any {
	return string(queue)
}

/*
* Retry
*  @Description:
* @param retries
* @return Option
 */
func Retry(retries int) OptionI {
	if retries < 0 {
		retries = 0
	}
	return retryOption(retries)
}
func (retry retryOption) String() string {
	return "retryOption"
}
func (retry retryOption) OptType() OptionType {
	return MaxRetryOpt
}
func (retry retryOption) Value() any {
	return int(retry)
}

/*
* Group
*  @Description:
* @param name
* @return Option
 */
func Group(name string) OptionI {
	return groupOption(name)
}
func (group groupOption) String() string {
	return "groupOption"
}
func (group groupOption) OptType() OptionType {
	return GroupOpt
}
func (group groupOption) Value() any {
	return string(group)
}

/*
* MaxLen
*  @Description:
* @param maxLen
* @return Option
 */
func MaxLen(maxLen int) OptionI {
	if maxLen < 0 {
		maxLen = 1000
	}
	return maxLenOption(maxLen)
}
func (ml maxLenOption) String() string {
	return "maxLenOption"
}
func (ml maxLenOption) OptType() OptionType {
	return MaxLenOpt
}
func (ml maxLenOption) Value() any {
	return int(ml)
}

/*
* ExecuteTime
*  @Description:
* @param tm
* @return Option
 */
func ExecuteTime(unixTime time.Time) OptionI {
	if unixTime.IsZero() {
		unixTime = time.Now()
	}
	return executeTime(unixTime)
}
func (et executeTime) String() string {
	return "executeTime"
}
func (et executeTime) OptType() OptionType {
	return ExecuteTimeOpt
}
func (et executeTime) Value() any {
	return time.Time(et)
}

/*
* Priority
*  @Description:
* @param priority
* @return OptionI
 */
func Priority(priority float64) OptionI {
	if priority > 10 {
		priority = 10
	}
	if priority < 0 {
		priority = 0
	}
	return priorityOption(priority)
}
func (pri priorityOption) String() string {
	return "priorityOption"
}
func (pri priorityOption) OptType() OptionType {
	return PriorityOpt
}
func (pri priorityOption) Value() any {
	return float64(pri)
}

/*
* composeOptions
*  @Description:
* @param options
* @return option
* @return error
 */
func ComposeOptions(options ...OptionI) (Option, error) {
	res := Option{
		Priority: DefaultOptions.Priority,
		Retry:    DefaultOptions.JobMaxRetry,
		Queue:    DefaultOptions.DefaultQueueName,
		Group:    DefaultOptions.DefaultGroup,
		MaxLen:   DefaultOptions.DefaultMaxLen,
	}
	for _, f := range options {
		switch f.OptType() {
		case PriorityOpt:
			if v, ok := f.Value().(float64); ok {
				res.Priority = v
			}
		case QueueOpt:
			if v, ok := f.Value().(string); ok {
				res.Queue = v
			}
		case GroupOpt:
			if v, ok := f.Value().(string); ok {
				res.Group = v
			}
		case MaxRetryOpt:
			if v, ok := f.Value().(int); ok {
				res.Retry = v
			}
		case MaxLenOpt:
			if v, ok := f.Value().(int64); ok {
				res.MaxLen = v
			}
		case ExecuteTimeOpt:
			if v, ok := f.Value().(time.Time); ok {
				res.ExecuteTime = v
			}
		}
	}
	return res, nil
}

//
//  ConsumerResult
//  @Description:

type FlagInfo string
type LevelMsg string

const (
	SuccessInfo FlagInfo = "success"
	FailedInfo  FlagInfo = "failed"

	ErrLevel  LevelMsg = "error"
	InfoLevel LevelMsg = "info"
)

type ConsumerResult struct {
	Level   LevelMsg
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
	Priority    float64

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

	Priority:         0,
	DefaultQueueName: "default-queue",
	DefaultGroup:     "default-group",
	DefaultMaxLen:    1000,

	DefaultDelayQueueName: "default-delay-queue",
	DefaultDelayGroup:     "default-delay-group",

	RetryTime: 800 * time.Millisecond,
	WorkCount: make(chan struct{}, 20),
}
