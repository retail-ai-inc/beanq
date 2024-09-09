package beanq

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/retail-ai-inc/beanq/helper/logger"
)

type WFMux interface {
	Name() string
	Value() string
	Until() time.Time
	LockContext(ctx context.Context) error
	UnlockContext(ctx context.Context) (bool, error)
	ExtendContext(ctx context.Context) (bool, error)
}

type Workflow struct {
	ctx                   context.Context
	gid                   string
	currentIndex          int
	currentTask           Task
	message               *Message
	tasks                 []Task
	results               []error
	rollbackResultHandler func(taskID string, err error) error
	trackRecordFunc       TaskRecordFunc
	wfMux                 WFMux
}

type TaskStatus int

type TaskRecordFunc func(gid string, taskID string, status TaskStatus, err error)

func (t TaskStatus) String() string {
	switch t {
	case ExecuteSuccess:
		return "execute success"
	case ExecuteFailed:
		return "execute failed"
	case RollbackSuccess:
		return "rollback success"
	case RollbackFailed:
		return "rollback failed"
	default:
		return strconv.Itoa(int(t))
	}
}

const (
	ExecuteSuccess TaskStatus = iota + 1
	ExecuteFailed
	RollbackSuccess
	RollbackFailed
	RollbackResultProcessFailed
)

func NewWorkflow(ctx context.Context, message *Message) *Workflow {
	return &Workflow{
		ctx:             ctx,
		gid:             strings.Join([]string{message.Channel, message.Topic, message.Id}, "-"),
		message:         message,
		tasks:           make([]Task, 0),
		results:         make([]error, 0),
		trackRecordFunc: nil,
	}
}

func (w *Workflow) SetTrackRecordFunc(fn TaskRecordFunc) {
	w.trackRecordFunc = fn
}

func (w *Workflow) SetMux(mux WFMux) {
	w.wfMux = mux
}

func (w *Workflow) SetGid(gid string) {
	w.gid = gid
}

func (w *Workflow) GetGid() string {
	return w.gid
}

// WithRollbackResultHandler handle rollback error
func (w *Workflow) WithRollbackResultHandler(handler func(taskID string, err error) error) *Workflow {
	w.rollbackResultHandler = handler
	return w
}

func (w *Workflow) Message() *Message {
	return w.message
}

func (w *Workflow) NewTask(ids ...string) *BaseTask {
	w.currentIndex++
	id := fmt.Sprintf("TASK-%d", w.currentIndex)
	if len(ids) > 0 {
		id = ids[0]
	}
	task := &BaseTask{
		id: id,
		wf: w,
	}
	w.tasks = append(w.tasks, task)
	w.results = make([]error, len(w.tasks))
	return task
}

func (w *Workflow) CurrentTask() Task {
	return w.currentTask
}

func (w *Workflow) TrackRecord(gid string, taskID string, status TaskStatus, err error) {
	if err == nil {
		err = errors.New(status.String())
	}
	logger.New().Error(fmt.Sprintf("workflow record: %s:%s:%d, memo: %v", gid, taskID, status, err))

	if w.trackRecordFunc != nil {
		w.trackRecordFunc(gid, taskID, status, err)
	}
}

func (w *Workflow) Run() (err error) {
	if w.wfMux != nil {
		if err := w.wfMux.LockContext(w.ctx); err != nil {
			return err
		}
		defer func() {
			if _, err := w.wfMux.UnlockContext(w.ctx); err != nil {
				logger.New().Error(err)
			}
		}()
	}
	for index, task := range w.tasks {
		func() {
			defer func() {
				if e := recover(); e != nil || err != nil {
					if err == nil {
						err = fmt.Errorf("%v", e)
					}
					w.results[index] = err

					w.TrackRecord(w.gid, task.ID(), ExecuteFailed, err)
					w.rollback(index)
				}
			}()

			w.currentTask = task
			if err = task.Execute(); err == nil {
				w.results[index] = nil
				w.TrackRecord(w.gid, task.ID(), ExecuteSuccess, nil)
			}
		}()

		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Workflow) rollback(from int) {
	for i := from - 1; i >= 0; i-- {
		err := func(index int) error {
			var err error
			task := w.tasks[index]

			defer func() {
				if e := recover(); e != nil || err != nil {
					// handle rollback error
					if err == nil {
						err = fmt.Errorf("%v", e)
					}

					w.TrackRecord(w.gid, task.ID(), RollbackFailed, err)

					if w.rollbackResultHandler != nil {
						err = w.rollbackResultHandler(task.ID(), err)
						if err != nil {
							w.TrackRecord(w.gid, task.ID(), RollbackResultProcessFailed, err)
						}
					}
				}
			}()

			err = task.Rollback()
			if err == nil {
				w.TrackRecord(w.gid, task.ID(), RollbackSuccess, nil)
			}
			return err
		}(i)
		if err != nil {
			// if meet some error when execute rollback func, should not continue the rollback process.
			break
		}
	}
}

func (w *Workflow) Results() []error {
	return w.results
}

type Task interface {
	ID() string
	Execute() error
	Rollback() error
}

// BaseTask ...
type BaseTask struct {
	id           string
	wf           *Workflow
	executeFunc  func(task Task) error
	rollbackFunc func(task Task) error
}

func (t *BaseTask) ID() string {
	return t.id
}

func (t *BaseTask) Execute() error {
	if t.executeFunc == nil {
		return errors.New("executeFunc is nil")
	}
	return t.executeFunc(t)
}

func (t *BaseTask) Rollback() error {
	if t.rollbackFunc == nil {
		return nil
	}
	return t.rollbackFunc(t)
}

func (t *BaseTask) OnExecute(fn func(task Task) error) *BaseTask {
	t.executeFunc = fn
	return t
}

func (t *BaseTask) OnRollback(fn func(task Task) error) *BaseTask {
	t.rollbackFunc = fn
	return t
}
