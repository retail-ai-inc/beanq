package beanq

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dtm-labs/client/dtmcli"
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
)

const (
	OpAction     = "action"
	OpCompensate = "compensate"
)

// ErrWorkflowFailure error of FAILURE
var ErrWorkflowFailure = errors.New("WORKFLOW FAILURE")

// ErrWorfflowOngoing error of ONGOING
var ErrWorkflowOngoing = errors.New("WORKFLOW ONGOING")

type (
	Workflow struct {
		gid              string
		alreadyExist     bool
		currentIndex     int
		ctx              context.Context
		wfMux            WFMux
		message          *Message
		onRollbackResult func(taskID string, err error)
		record           *WorkflowRecord
		tasks            tasks
		transStore       TransStore
		transaction      *TransGlobal
		progresses       []TransBranch
		steps            map[string]*branchResult
	}
	WFMux interface {
		Name() string
		Value() string
		Until() time.Time
		LockContext(ctx context.Context) error
		UnlockContext(ctx context.Context) (bool, error)
		ExtendContext(ctx context.Context) (bool, error)
	}

	branchResult struct {
		taskID   string
		branchID string
		status   string
		started  bool
		op       string
		err      error
	}
)

var (
	workflowClient redis.UniversalClient
	workflowConfig Redis
)

// make workflow as an independent module
func InitWorkflow(config *BeanqConfig) {
	redisConfig := config.Redis
	workflowClient = bredis.NewRdb(
		redisConfig.Host,
		redisConfig.Port,
		redisConfig.Password,
		redisConfig.Database,
		redisConfig.MaxRetries,
		redisConfig.DialTimeout,
		redisConfig.ReadTimeout,
		redisConfig.WriteTimeout,
		redisConfig.PoolTimeout,
		redisConfig.PoolSize,
		redisConfig.MinIdleConnections)

	workflowConfig = redisConfig
}

func NewWorkflow(ctx context.Context, message *Message) (*Workflow, error) {
	gid := strings.Join([]string{message.Channel, message.Topic, message.Id}, "-")
	// prepare workflow, get process from redis by gid
	ts := NewTransStore(
		workflowClient,
		workflowConfig.Prefix+":"+"workflow",
		7*24*time.Hour)

	transGlobal := &TransGlobal{
		Message: message,
	}
	err := transGlobal.New()
	if err != nil {
		return nil, errorstack.WithStack(err)
	}

	err = ts.MaySaveNew(ctx, transGlobal, nil)

	var transBranch []TransBranch

	var alreadyExist bool
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
		ctx:          ctx,
		gid:          gid,
		message:      message,
		tasks:        make([]task, 0),
		record:       NewWorkflowRecord(),
		transStore:   ts,
		transaction:  transGlobal,
		progresses:   transBranch,
		alreadyExist: alreadyExist,
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
func (w *Workflow) OnRollbackResult(handler func(taskID string, err error)) *Workflow {
	w.onRollbackResult = handler
	return w
}

func (w *Workflow) Message() *Message {
	return w.message
}

func (w *Workflow) NewTask(ids ...string) *task {
	w.currentIndex++

	id := fmt.Sprintf("TASK-%d", w.currentIndex)
	if len(ids) > 0 {
		id = ids[0]
	}

	t := task{
		id: id,
		wf: w,
	}

	w.tasks = append(w.tasks, t)
	return &t
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

	// save branch result
	err = w.initProgresses()
	if err != nil {
		return err
	}
	actions, compensates := w.initSteps()

	if w.transaction.Status == StatusSubmitted {
		for _, branch := range actions() {
			func() {
				defer func() {
					if e := recover(); e != nil || err != nil {
						if err == nil {
							err = fmt.Errorf("%v", e)
						}
					}
				}()

				err = w.tasks.Run(w.ctx, &branch, OpAction)
				if err != nil {
					err2 := w.ChangeStatus(w.ctx, dtmcli.StatusAborting, withRollbackReason(err.Error()))
					if err2 != nil {
						err = errorstack.Wrap(err, err2.Error())
					}
				}
			}()
		}

		if err == nil {
			err2 := w.ChangeStatus(w.ctx, dtmcli.StatusSucceed)
			if err2 != nil {
				return err2
			}
		}
	}

	if w.transaction.Status == StatusAborting {
		for _, branch := range compensates() {
			func() {
				defer func() {
					if e := recover(); e != nil || err != nil {
						if err == nil {
							err = fmt.Errorf("%v", e)
						}
					}
				}()

				err = w.tasks.Run(w.ctx, &branch, OpCompensate)
				if err != nil {
					err2 := w.ChangeStatus(w.ctx, dtmcli.StatusAborting, withRollbackReason(err.Error()))
					if err2 != nil {
						err = errorstack.Wrap(err, err2.Error())
					}
				}

				if w.onRollbackResult != nil {
					w.onRollbackResult(branch.TaskID, err)
				}
			}()
		}

		if err == nil {
			err2 := w.ChangeStatus(w.ctx, dtmcli.StatusFailed)
			if err2 != nil {
				return err2
			}
		}
	}

	return nil
}

func (w *Workflow) ChangeStatus(ctx context.Context, status string, opts ...changeStatusOption) error {
	statusParams := &changeStatusParams{}
	for _, opt := range opts {
		opt(statusParams)
	}

	updates := []string{"status", "update_time"}
	now := time.Now()
	if status == dtmcli.StatusSucceed {
		w.transaction.FinishTime = &now
		updates = append(updates, "finish_time")
	} else if status == dtmcli.StatusFailed {
		w.transaction.RollbackTime = &now
		updates = append(updates, "rollback_time")
	}

	if statusParams.rollbackReason != "" {
		w.transaction.RollbackReason = statusParams.rollbackReason
		updates = append(updates, "rollback_reason")
	}

	if statusParams.result != "" {
		w.transaction.Result = statusParams.result
		updates = append(updates, "result")
	}

	w.transaction.UpdateTime = &now
	err := w.transStore.ChangeGlobalStatus(ctx, w.transaction, status, updates, status == StatusSucceed || status == StatusFailed, -1)
	if err != nil {
		return err
	}
	w.transaction.Status = status

	return nil
}

func (w *Workflow) initProgresses() error {
	if w.alreadyExist {
		return nil
	}

	now := time.Now()

	var progresses []TransBranch
	var index int

	for i, task := range w.tasks {
		branchID := fmt.Sprintf("%02d", i+1)

		for _, op := range []string{OpCompensate, OpAction} {
			index++
			progresses = append(progresses, TransBranch{
				Index:        index,
				Gid:          w.gid,
				Statement:    string(task.Statement()),
				BinData:      []byte{},
				BranchID:     branchID,
				TaskID:       task.ID(),
				Op:           op,
				Status:       StatusPrepared,
				FinishTime:   nil,
				RollbackTime: nil,
				Error:        nil,
				CreateTime:   &now,
				UpdateTime:   &now,
			})
		}
	}

	err := w.transStore.LockGlobalSaveBranches(w.ctx, w.gid, StatusPrepared, progresses, -1)
	if err != nil {
		return errorstack.WithStack(err)
	}

	w.progresses = progresses
	return nil
}

func (w *Workflow) initSteps() (actions, compensates func() []TransBranch) {
	n := len(w.progresses)
	branchResults := w.progresses

	shouldRun := func(current int) bool {
		// check the branch in previous step is succeed
		if current >= 2 && branchResults[current-2].Status != StatusSucceed {
			return false
		}

		return true
	}

	shouldRollback := func(current int) bool {
		rollbacked := func(i int) bool {
			// current compensate op rollbacked or related action still prepared
			return branchResults[i].Status == StatusSucceed || branchResults[i+1].Status == StatusPrepared
		}
		if rollbacked(current) {
			return false
		}
		// if !csc.Concurrent，then check the branch in next step is rollbacked
		if current < n-2 && !rollbacked(current+2) {
			return false
		}

		return true
	}

	pickToRunActions := func() []TransBranch {
		var toRun []TransBranch
		for current := 1; current < n; current += 2 {
			br := &branchResults[current]
			if br.Status == dtmcli.StatusPrepared && shouldRun(current) {
				toRun = append(toRun, *br)
			}
		}

		return toRun
	}

	pickToRunCompensates := func() []TransBranch {
		var toRun []TransBranch
		for current := n - 2; current >= 0; current -= 2 {
			br := &branchResults[current]
			if br.Status == StatusPrepared && shouldRollback(current) {
				toRun = append(toRun, *br)
			}
		}
		return toRun
	}

	return pickToRunActions, pickToRunCompensates
}

type Task interface {
	ID() string
	Execute() error
	Rollback() error
	Statement() []byte
	UpdateStatus(ctx context.Context, branch *TransBranch, oerr error) error
}

type tasks []task

func (t tasks) Run(ctx context.Context, branch *TransBranch, op string) error {
	var err error
	var tk *task
	taskID := branch.TaskID
	for _, v := range t {
		if v.id == taskID {
			tk = &v
			break
		}
	}

	if tk == nil {
		return errorstack.New("no task to run")
	}

	switch op {
	case OpAction:
		err = tk.Execute()
	case OpCompensate:
		err = tk.Rollback()
	default:
		err = errorstack.New("unspport task option")
	}

	err = tk.UpdateStatus(ctx, branch, err)
	if err != nil {
		return err
	}

	return nil
}

// task ...
type task struct {
	id           string
	wf           *Workflow
	executeFunc  func(task Task) error
	rollbackFunc func(task Task) error
	statement    []byte
	status       string
}

func (t *task) ID() string {
	return t.id
}

func (t *task) Execute() error {
	if t.executeFunc == nil {
		return errors.New("executeFunc is nil")
	}

	// business side handle the retry
	err := t.executeFunc(t)

	status := ExecuteSuccess
	if err != nil {
		status = ExecuteFailed
	}

	t.trackRecord(t.id, &TaskStatus{
		status:    status,
		statement: t.Statement(),
		err:       err,
	})

	return err
}

func (t *task) Rollback() error {
	if t.rollbackFunc == nil {
		return nil
	}
	err := t.rollbackFunc(t)

	status := RollbackSuccess
	if err != nil {
		status = RollbackFailed
	}

	t.trackRecord(t.id, &TaskStatus{
		status:    status,
		statement: t.Statement(),
		err:       err,
	})

	return err
}

func (t *task) OnExecute(fn func(task Task) error) *task {
	t.executeFunc = fn
	return t
}

func (t *task) OnRollback(fn func(task Task) error) *task {
	t.rollbackFunc = fn
	return t
}

func (t *task) WithRecordStatement(statement []byte) *task {
	t.statement = statement
	return t
}

func (t *task) Statement() []byte {
	return t.statement
}

func (t *task) trackRecord(taskID string, status *TaskStatus) {
	w := t.wf
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

func (t *task) UpdateStatus(ctx context.Context, branch *TransBranch, oerr error) error {
	var status string
	switch {
	case oerr == nil:
		status = StatusSucceed
	case branch.Op == OpAction && errors.Is(oerr, ErrWorkflowFailure):
		branch.Error = fmt.Errorf("return failed: %w", oerr)
		status = dtmcli.StatusFailed
	case errors.Is(oerr, ErrWorkflowOngoing):
		status = ""
	default:
	}

	if status != "" {
		now := time.Now()
		branch.FinishTime = &now
		branch.UpdateTime = &now
		branch.Status = status
		err := t.wf.transStore.LockGlobalSaveBranches(ctx, t.wf.transaction.Gid, status, []TransBranch{*branch}, branch.Index)
		if err != nil {
			return errorstack.Wrap(err, oerr.Error())
		}
	}

	return errorstack.WithStack(oerr)
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

type changeStatusParams struct {
	rollbackReason string
	result         string
}

type changeStatusOption func(c *changeStatusParams)

func withRollbackReason(rollbackReason string) changeStatusOption {
	return func(c *changeStatusParams) {
		c.rollbackReason = rollbackReason
	}
}

func withResult(result string) changeStatusOption {
	return func(c *changeStatusParams) {
		c.result = result
	}
}
