package beanq

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/retail-ai-inc/beanq/v3/internal/driver/bredis"

	errorstack "github.com/pkg/errors"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
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

var ErrWorkflowFailure = errors.New("WORKFLOW FAILURE")

type (
	Workflow struct {
		ctx              context.Context
		currentTask      Task
		wfMux            WFMux
		message          *Message
		onRollbackResult func(taskID string, err error) error
		record           *WorkflowRecord
		gid              string
		tasks            []Task
		results          []error
		currentIndex     int

		transaction *TransGlobal
		progresses  []TransBranch
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

var (
	workflowClient redis.UniversalClient
	workflowConfig Redis
)

// make workflow as an independent module
func InitWorkflow(config *BeanqConfig) {
	workflowConfig = config.Redis
	workflowClient = bredis.NewRdb(
		workflowConfig.Host,
		workflowConfig.Port,
		workflowConfig.Password,
		workflowConfig.Database,
		workflowConfig.MaxRetries,
		workflowConfig.DialTimeout,
		workflowConfig.ReadTimeout,
		workflowConfig.WriteTimeout,
		workflowConfig.PoolTimeout,
		workflowConfig.PoolSize,
		workflowConfig.MinIdleConnections)
}

func NewWorkflow(ctx context.Context, message *Message) (*Workflow, error) {
	gid := strings.Join([]string{message.Channel, message.Topic, message.Id}, "-")
	// prepare workflow, get process from redis by gid
	ts := NewTransStore(workflowClient, "workflow")
	transGlobal := &TransGlobal{
		Message: message,
	}
	err := transGlobal.New()
	if err != nil {
		return nil, errorstack.WithStack(err)
	}

	err = ts.MaySaveNew(ctx, transGlobal, nil)

	var transBranch []TransBranch

	if errors.Is(err, ErrUniqueConflict) {
		// if exist, get global and branch trans info from redis
		transGlobal, err = ts.FindGlobal(ctx, gid)
		if err != nil {
			return nil, errorstack.WithStack(err)
		}

		transBranch, err = ts.FindBranches(ctx, gid)
		if err != nil {
			return nil, errorstack.WithStack(err)
		}
	}

	return &Workflow{
		ctx:         ctx,
		gid:         gid,
		message:     message,
		tasks:       make([]Task, 0),
		results:     make([]error, 0),
		record:      NewWorkflowRecord(),
		transaction: transGlobal,
		progresses:  transBranch,
	}, nil
}

func (w *Workflow) Init(opts ...func(workflow *Workflow)) {
	for _, opt := range opts {
		opt(w)
	}
}

func WfOptionRecordErrorHandler(handler func(error)) func(workflow *Workflow) {
	return func(workflow *Workflow) {
		if workflow.record != nil {
			workflow.record.setErrorHandler(handler)
		}
	}
}

func WfOptionMux(mux WFMux) func(workflow *Workflow) {
	return func(workflow *Workflow) {
		workflow.wfMux = mux
	}
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
	data := struct {
		CreatedAt time.Time          `bson:"CreatedAt"`
		UpdatedAt time.Time          `bson:"UpdatedAt"`
		Channel   string             `bson:"Channel"`
		Topic     string             `bson:"Topic"`
		MessageID string             `bson:"MessageId"`
		GID       string             `bson:"Gid"`
		TaskID    string             `bson:"TaskId"`
		Status    string             `bson:"Status"`
		Statement string             `bson:"Statement"`
		Id        primitive.ObjectID `bson:"_id"`
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
	switch w.transaction.Status {
	case StatusSucceed:
		// already success
		return nil
	case StatusFailed:
		return errorstack.Wrap(ErrWorkflowFailure, w.transaction.RollbackReason)
	default:
	}

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

	// TODO save branch result 

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
	errorHandler    func(error)
	mongoCollection *mongo.Collection
	asyncPool       *asyncPool
	retry           int
	on              bool
	async           bool
}

var (
	workflowRecordOnce sync.Once
	workflowRecord     *WorkflowRecord
)

func NewWorkflowRecord() *WorkflowRecord {
	var config struct {
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
		Retry int
		On    bool
		Async bool
	}
	workflowRecordOnce.Do(func() {
		err := viper.UnmarshalKey("queue.workflow.record", &config)
		if err != nil {
			logger.New().Info("no workflow configration, ignoring the record")
		}
		fmt.Printf("workflow:%+v \n", config)
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
			return w.SyncWrite(c, data)
		})
		return
	}

	w.errorHandler(w.SyncWrite(ctx, data))
}

func (w *WorkflowRecord) SyncWrite(ctx context.Context, data any) error {
	if !w.on || w.mongoCollection == nil {
		logger.New().Info(fmt.Sprintf("workflow record data: %+v", data))
		return nil
	}

	for i := 0; i <= w.retry; i++ {
		_, err := w.mongoCollection.InsertOne(ctx, data)
		if err == nil {
			return nil
		}
		if i == w.retry {
			return fmt.Errorf("[workflow recored] write error: %w", err)
		}

		waitTime := tool.JitterBackoff(500*time.Millisecond, time.Second, i)
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return fmt.Errorf("[workflow recored] context error: %w", ctx.Err())
		}
	}
	return nil
}

type TaskStatus struct {
	err       error
	statement []byte
	status    int
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
