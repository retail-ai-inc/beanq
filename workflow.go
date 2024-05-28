package beanq

import (
	"errors"
	"fmt"
)

type Workflow struct {
	currentIndex          int
	currentTask           Task
	message               *Message
	tasks                 []Task
	results               []error
	rollbackResultHandler func(taskID string, err error)
}

func NewWorkflow(message *Message) *Workflow {
	return &Workflow{
		message: message,
		tasks:   make([]Task, 0),
		results: make([]error, 0),
	}
}

// WithRollbackResultHandler handle rollback error
func (w *Workflow) WithRollbackResultHandler(handler func(taskID string, err error)) *Workflow {
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

func (w *Workflow) Run() (err error) {
	for index, task := range w.tasks {
		func() {
			defer func() {
				if e := recover(); e != nil || err != nil {
					w.rollback(index)
					if err == nil {
						err = fmt.Errorf("%v", e)
					}
					w.results[index] = err
				}
			}()

			w.currentTask = task
			if err = task.Execute(); err == nil {
				w.results[index] = nil
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
		func(index int) {
			var err error
			task := w.tasks[index]

			defer func() {
				if e := recover(); e != nil || err != nil {
					// handle rollback error
					if err == nil {
						err = fmt.Errorf("%v", e)
					}

					if w.rollbackResultHandler != nil {
						w.rollbackResultHandler(task.ID(), err)
					}
				}
			}()

			err = task.Rollback()
		}(i)
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
