package beanq

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	errorstack "github.com/pkg/errors"
	"github.com/retail-ai-inc/beanq/helper/logger"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	ExecuteSuccess = iota + 1
	ExecuteFailed
	RollbackSuccess
	RollbackFailed
	RollbackResultProcessFailed
)

type (
	Workflow struct {
		ctx              context.Context
		gid              string
		currentIndex     int
		currentTask      Task
		message          *Message
		tasks            []Task
		results          []error
		onRollbackResult func(taskID string, err error) error
		wfMux            WFMux
		record           *WorkflowRecord
	}
	WFMux interface {
		Name() string
		Value() string
		Until() time.Time
		LockContext(ctx context.Context) error
		UnlockContext(ctx context.Context) (bool, error)
		ExtendContext(ctx context.Context) (bool, error)
	}
)

func NewWorkflow(ctx context.Context, message *Message) *Workflow {
	return &Workflow{
		ctx:     ctx,
		gid:     strings.Join([]string{message.Channel, message.Topic, message.Id}, "-"),
		message: message,
		tasks:   make([]Task, 0),
		results: make([]error, 0),
		record:  NewWorkflowRecord(),
	}
}

func (w *Workflow) SetRecordErrorHandler(handler func(error)) {
	if w.record != nil {
		w.record.setErrorHandler(handler)
	}
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

// OnRollbackResult handle rollback error
func (w *Workflow) OnRollbackResult(handler func(taskID string, err error) error) *Workflow {
	w.onRollbackResult = handler
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

func (w *Workflow) TrackRecord(taskID string, status TaskStatus) {
	var data = struct {
		Id        primitive.ObjectID `bson:"_id"`
		Channel   string             `bson:"Channel"`
		Topic     string             `bson:"Topic"`
		MessageID string             `bson:"MessageId"`
		GID       string             `bson:"Gid"`
		TaskID    string             `bson:"TaskId"`
		Status    string             `bson:"Status"`
		Statement string             `bson:"Statement"`
		CreatedAt time.Time          `bson:"CreatedAt"`
		UpdatedAt time.Time          `bson:"UpdatedAt"`
	}{
		Id:        primitive.NewObjectID(),
		Channel:   w.message.Channel,
		Topic:     w.message.Topic,
		MessageID: w.message.Id,
		GID:       w.gid,
		TaskID:    taskID,
		Status:    status.Status(),
		Statement: status.Statement(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.record.Write(w.ctx, data)
}

func (w *Workflow) Run() (err error) {
	if w.wfMux != nil {
		if err := w.wfMux.LockContext(w.ctx); err != nil {
			return errorstack.WithStack(err)
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
					w.TrackRecord(task.ID(), TaskStatus{
						status:    ExecuteFailed,
						statement: task.Statement(),
						err:       err,
					})
					w.rollback(index)
				}
			}()

			w.currentTask = task
			if err = task.Execute(); err == nil {
				w.results[index] = nil
				w.TrackRecord(task.ID(), TaskStatus{
					status:    ExecuteSuccess,
					statement: task.Statement(),
					err:       nil,
				})
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

					w.TrackRecord(task.ID(), TaskStatus{
						status:    RollbackFailed,
						statement: task.Statement(),
						err:       err,
					})

					if w.onRollbackResult != nil {
						err = w.onRollbackResult(task.ID(), err)
						if err != nil {
							w.TrackRecord(task.ID(), TaskStatus{
								status:    RollbackResultProcessFailed,
								statement: task.Statement(),
								err:       err,
							})
						}
					}
				}
			}()

			err = task.Rollback()
			if err == nil {
				w.TrackRecord(task.ID(), TaskStatus{
					status:    RollbackSuccess,
					statement: task.Statement(),
					err:       nil,
				})
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
	Statement() []byte
}

// BaseTask ...
type BaseTask struct {
	id           string
	wf           *Workflow
	executeFunc  func(task Task) error
	rollbackFunc func(task Task) error
	statement    []byte
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

func (t *BaseTask) WithRecordStatement(statement []byte) *BaseTask {
	t.statement = statement
	return t
}

func (t *BaseTask) Statement() []byte {
	return t.statement
}

type WorkflowRecord struct {
	on              bool
	retry           int
	async           bool
	errorHandler    func(error)
	mongoCollection *mongo.Collection
	asyncPool       *asyncPool
}

var workflowRecordOnce sync.Once
var workflowRecord *WorkflowRecord

func NewWorkflowRecord() *WorkflowRecord {
	var config struct {
		On    bool
		Retry int
		Async bool
		Mongo *struct {
			Database              string
			Collection            string
			UserName              string
			Password              string
			Host                  string
			Port                  string
			ConnectTimeOut        time.Duration
			MaxConnectionPoolSize uint64
			MaxConnectionLifeTime time.Duration
		}
	}
	workflowRecordOnce.Do(func() {
		err := viper.UnmarshalKey("queue.workflow.record", &config)
		if err != nil {
			logger.New().Info("no workflow configration, ignoring the record")
		}

		workflowRecord = &WorkflowRecord{
			on:    config.On,
			retry: config.Retry,
			async: config.Async,
			errorHandler: func(err error) {
				logger.New().Error(err)
			},
			asyncPool: newAsyncPool(-1),
		}

		if config.On && config.Mongo != nil && config.Mongo.Database != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			connURI := "mongodb://" + config.Mongo.Host + ":" + config.Mongo.Port
			opts := options.Client().
				ApplyURI(connURI).
				SetConnectTimeout(config.Mongo.ConnectTimeOut).
				SetMaxPoolSize(config.Mongo.MaxConnectionPoolSize).
				SetMaxConnIdleTime(config.Mongo.MaxConnectionLifeTime)

			if config.Mongo.UserName != "" && config.Mongo.Password != "" {
				opts.SetAuth(options.Credential{
					AuthSource: config.Mongo.Database,
					Username:   config.Mongo.UserName,
					Password:   config.Mongo.Password,
				})
			}

			mdb, err := mongo.Connect(ctx, opts)
			if err != nil {
				panic(err)
			}

			// check connect
			err = mdb.Ping(ctx, nil)
			if err != nil {
				panic(err)
			}
			workflowRecord.mongoCollection = mdb.Database(config.Mongo.Database).Collection(config.Mongo.Collection)
		}
	})
	return workflowRecord
}

func (w *WorkflowRecord) setErrorHandler(handler func(error)) {
	w.errorHandler = handler
	w.asyncPool.captureException = func(ctx context.Context, err any) {
		handler(fmt.Errorf("%+v", err))
	}
}

func (w *WorkflowRecord) Write(ctx context.Context, data any) {
	if w.async {
		w.asyncPool.Execute(context.Background(), func(c context.Context) error {
			w.SyncWrite(c, data)
			return nil
		})
		return
	}

	w.SyncWrite(ctx, data)
}

func (w *WorkflowRecord) SyncWrite(ctx context.Context, data any) {
	if !w.on || w.mongoCollection == nil {
		logger.New().Info(fmt.Sprintf("workflow record data: %+v", data))
		return
	}

	for i := 0; i <= w.retry; i++ {
		_, err := w.mongoCollection.InsertOne(ctx, data)
		if err == nil {
			return
		}
		if i == w.retry {
			w.errorHandler(fmt.Errorf("[workflow recored] write error: %w", err))
			return
		}

		waitTime := jitterBackoff(500*time.Millisecond, time.Second, i)
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			w.errorHandler(fmt.Errorf("[workflow recored] context error: %w", ctx.Err()))
			return
		}
	}
}

type TaskStatus struct {
	status    int
	statement []byte
	err       error
}

func (t *TaskStatus) Status() string {
	switch t.status {
	case ExecuteSuccess:
		return "execute success"
	case ExecuteFailed:
		return "execute failed"
	case RollbackSuccess:
		return "rollback success"
	case RollbackFailed:
		return "rollback failed"
	default:
		return strconv.Itoa(t.status)
	}
}

func (t *TaskStatus) String() string {
	return fmt.Sprintf("[status:%s; statement:%s]\n", t.Status(), t.Statement())
}

func (t *TaskStatus) Error() error {
	return t.err
}

func (t *TaskStatus) Statement() string {
	return string(t.statement)
}
