package beanq

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	errorstack "github.com/pkg/errors"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/internal/driver/bredis"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
		skipper          func(error) bool
	}

	WFMux interface {
		Name() string
		Value() string
		Until() time.Time
		LockContext(ctx context.Context) error
		UnlockContext(ctx context.Context) (bool, error)
		ExtendContext(ctx context.Context) (bool, error)
	}

	WorkFlow struct {
		On      bool   `json:"on"`
		Retry   int    `json:"retry"`
		Async   bool   `json:"async"`
		Storage string `json:"storage"`
	}
)

var (
	workflowClient      redis.UniversalClient
	workflowRedisConfig *Redis
	workflowOnce        sync.Once
	workflowConfig      = &struct {
		*Mongo
		*WorkFlow
		*History
		Collection struct {
			Name  string
			Shard bool
		}
	}{}
)

// InitWorkflow make workflow as an independent module
func InitWorkflow(beanqConfig *BeanqConfig) {
	workflowOnce.Do(func() {
		workflowClient = bredis.NewRdb(
			beanqConfig.Redis.Host,
			beanqConfig.Redis.Port,
			beanqConfig.Redis.Password,
			beanqConfig.Redis.Database,
			beanqConfig.Redis.MaxRetries,
			beanqConfig.Redis.DialTimeout,
			beanqConfig.Redis.ReadTimeout,
			beanqConfig.Redis.WriteTimeout,
			beanqConfig.Redis.PoolTimeout,
			beanqConfig.Redis.PoolSize,
			beanqConfig.Redis.MinIdleConnections)

		workflowRedisConfig = &beanqConfig.Redis
		workflowConfig.Collection = struct {
			Name  string
			Shard bool
		}{Name: "workflow_records", Shard: true}
		//nolint:staticcheck,qf1008 //enhance readability
		if v, ok := beanqConfig.Mongo.Collections["workflow"]; ok {
			workflowConfig.Collection.Name = v.Name
			workflowConfig.Collection.Shard = v.Shard
		}
		workflowConfig.WorkFlow = &beanqConfig.WorkFlow
		workflowConfig.History = &beanqConfig.History
		workflowConfig.Mongo = beanqConfig.Mongo
	})
}

func NewWorkflow(ctx context.Context, message *Message) (*Workflow, error) {
	if workflowClient == nil {
		panic("workflow client not initialized")
	}

	// prepare workflow, get process from redis by gid
	ts := NewTransStore(
		workflowClient,
		workflowRedisConfig.Prefix+":"+"workflow",
		7*24*time.Hour)

	transGlobal, err := NewTransGlobal(message)
	if err != nil {
		return nil, errorstack.WithStack(err)
	}

	return &Workflow{
		ctx:         ctx,
		gid:         message.Id,
		message:     message,
		tasks:       make([]*task, 0),
		record:      NewWorkflowRecord(),
		transStore:  ts,
		transaction: transGlobal,
		progresses:  []TransBranch{},
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

func WfSkipper(skipper func(error) bool) func(workflow *Workflow) {
	return func(worflow *Workflow) {
		worflow.skipper = skipper
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

	skipper := func(error) bool {
		return false
	}

	if w.skipper != nil {
		skipper = w.skipper
	}

	t := &task{
		id:      id,
		wf:      w,
		skipper: skipper,
	}

	w.tasks = append(w.tasks, t)
	return t
}

func (w *Workflow) Run() (err error) {
	var progresses []TransBranch
	index := -1
	now := time.Now()

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
				Error:        "",
				CreateTime:   &now,
				UpdateTime:   &now,
			})
		}
	}

	err = w.transStore.MaySaveNew(w.ctx, w.transaction, progresses)

	if errors.Is(err, ErrUniqueConflict) {
		// if exist, get global and branch trans info from redis
		w.transaction, err = w.transStore.FindGlobal(w.ctx, w.gid)
		if err != nil {
			return errorstack.WithStack(err)
		}

		w.progresses, err = w.transStore.FindBranches(w.ctx, w.gid)
		if err != nil {
			return errorstack.WithStack(err)
		}
	} else {
		w.progresses = progresses
	}

	switch w.transaction.Status {
	case StatusSucceed:
		// already success
		return nil
	case StatusFailed:
		return errorstack.Wrap(ErrWorkflowFailure, w.transaction.Reason)
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

	actions, compensates := w.initSteps()

ACTION:
	for w.transaction.Status == StatusPrepared {
		select {
		case <-w.ctx.Done():
			err = w.ctx.Err()
			break ACTION
		default:
		}

		ac := actions()
		if len(ac) == 0 {
			break ACTION
		}

		for _, branch := range ac {
			err = w.executor(branch, OpAction)
			if err != nil {
				break ACTION
			}
		}
	}

	if err == nil {
		err2 := w.ChangeStatus(w.ctx, StatusSucceed)
		if err2 != nil {
			return err2
		}
		return nil
	}

COMPENSATE:
	for w.transaction.Status == StatusAborting {
		select {
		case <-w.ctx.Done():
			err = w.ctx.Err()
			break COMPENSATE
		default:
		}

		ac := compensates()
		if len(ac) == 0 {
			break COMPENSATE
		}

		for _, branch := range ac {
			err = w.executor(branch, OpCompensate)
			if err != nil {
				break COMPENSATE
			}
		}
	}

	if err == nil {
		err2 := w.ChangeStatus(w.ctx, StatusFailed)
		if err2 != nil {
			return err2
		}
		return nil
	}

	return nil
}

func (w *Workflow) executor(branch *TransBranch, op string) (err error) {
	defer func() {
		if e := recover(); e != nil || err != nil {
			if err == nil {
				logger.New().Panic(fmt.Sprintf("%v\n%s", e, string(debug.Stack())))
				err = fmt.Errorf("%v", e)
			}
		}
	}()

	err = w.tasks.Run(w.ctx, branch, op)
	if err != nil {
		err2 := w.ChangeStatus(w.ctx, StatusAborting, err.Error())
		if err2 != nil {
			err = errorstack.Wrap(err, err2.Error())
		}
	}

	if op == OpCompensate && w.onRollbackResult != nil {
		w.onRollbackResult(branch.TaskID, err)
	}

	return err
}

func (w *Workflow) ChangeStatus(ctx context.Context, status string, reason ...string) error {
	updates := []string{"status", "update_time"}
	now := time.Now()
	switch status {
	case StatusSucceed:
		w.transaction.FinishTime = &now
		updates = append(updates, "finish_time")
	case StatusFailed:
		w.transaction.RollbackTime = &now
		updates = append(updates, "rollback_time")
	}

	if len(reason) > 0 && reason[0] != "" {
		w.transaction.Reason = reason[0]
		updates = append(updates, "reason")
	}

	w.transaction.UpdateTime = &now
	err := w.transStore.ChangeGlobalStatus(ctx, w.transaction, status, updates, status == StatusSucceed || status == StatusFailed, -1)
	if err != nil {
		return err
	}

	return nil
}

func (w *Workflow) initSteps() (actions, compensates func() []*TransBranch) {
	n := len(w.progresses)
	branchResults := make([]*TransBranch, n)
	for i := range w.progresses {
		branchResults[i] = &w.progresses[i]
	}

	shouldRun := func(current int) bool {
		if branchResults[current].Status != StatusPrepared {
			return false
		}

		// check the branch in previous step is succeed
		if current >= 2 && branchResults[current-2].Status != StatusSucceed {
			return false
		}

		return true
	}

	shouldRollback := func(current int) bool {
		rollbacked := func(i int) bool {
			// current compensate op rollbacked or related action still prepared
			return branchResults[i].Status == StatusSucceed ||
				branchResults[i+1].Status == StatusPrepared
		}

		if branchResults[current].Status != StatusPrepared {
			return false
		}

		if rollbacked(current) {
			return false
		}

		// check the branch in next step is rollbacked
		if current < n-2 && !rollbacked(current+2) {
			return false
		}

		return true
	}

	pickToRunActions := func() []*TransBranch {
		var toRun []*TransBranch
		for current := 1; current < n; current += 2 {
			if shouldRun(current) {
				toRun = append(toRun, branchResults[current])
			}
		}

		return toRun
	}

	pickToRunCompensates := func() []*TransBranch {
		var toRun []*TransBranch
		for current := n - 2; current >= 0; current -= 2 {
			br := branchResults[current]
			if shouldRollback(current) {
				toRun = append(toRun, br)
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

type tasks []*task

func (t tasks) Run(ctx context.Context, branch *TransBranch, op string) error {
	var err error
	var tk *task
	taskID := branch.TaskID
	for _, v := range t {
		if v.id == taskID {
			tk = v
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
	skipper      func(error) bool
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

	t.trackRecord(t.id, &TaskStatus{
		option:    OpAction,
		statement: t.Statement(),
		err:       err,
	})

	return err
}

func (t *task) Skipper(skipper func(error) bool) *task {
	t.skipper = skipper
	return t
}

func (t *task) Rollback() error {
	if t.rollbackFunc == nil {
		return nil
	}
	err := t.rollbackFunc(t)

	t.trackRecord(t.id, &TaskStatus{
		option:    OpCompensate,
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
	var skipper bool
	w := t.wf
	st := StatusSucceed
	if status.err != nil {
		st = StatusFailed
		skipper = t.skipper(status.err)
	}

	data := struct {
		CreatedAt time.Time          `bson:"CreatedAt"`
		UpdatedAt time.Time          `bson:"UpdatedAt"`
		Channel   string             `bson:"Channel"`
		Topic     string             `bson:"Topic"`
		GID       string             `bson:"Gid"`
		TaskID    string             `bson:"TaskId"`
		Option    string             `bson:"Option"`
		Status    string             `bson:"Status"`
		Statement string             `bson:"Statement"`
		Error     string             `bson:"Error"`
		Skipper   bool               `bson:"Skipper"`
		Id        primitive.ObjectID `bson:"_id"`
	}{
		Id:        primitive.NewObjectID(),
		Channel:   w.message.Channel,
		Topic:     w.message.Topic,
		GID:       w.gid,
		TaskID:    taskID,
		Option:    status.option,
		Status:    st,
		Statement: status.Statement(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Error:     status.Error(),
		Skipper:   skipper,
	}

	w.record.Write(w.ctx, data)
}

func (t *task) UpdateStatus(ctx context.Context, branch *TransBranch, oerr error) error {
	// status only has two type: succeed or failed
	var status string

	switch {
	case oerr == nil:
		status = StatusSucceed
	case t.skipper(oerr):
		status = StatusSucceed
		branch.Error = oerr.Error()
		oerr = nil
	case errors.Is(oerr, ErrWorkflowOngoing):
		status = ""
		branch.Error = oerr.Error()
	default:
		status = StatusFailed
		branch.Error = oerr.Error()
	}

	if status != "" {
		now := time.Now()
		branch.FinishTime = &now
		branch.UpdateTime = &now
		branch.Status = status

		err := t.wf.transStore.LockGlobalSaveBranches(ctx, t.wf.transaction.Gid, t.wf.transaction.Status, []TransBranch{*branch}, branch.Index)
		if err != nil {
			return errorstack.WithStack(err)
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
	workflowRecordOnce.Do(func() {

		mongoCfg := workflowConfig.Mongo
		workflowCfg := workflowConfig.WorkFlow
		collection := workflowConfig.Collection

		workflowRecord = &WorkflowRecord{
			on:    workflowCfg.On,
			retry: workflowCfg.Retry,
			async: workflowCfg.Async,
			errorHandler: func(err error) {
				if err == nil {
					return
				}
				logger.New().Error(err)
			},
			asyncPool: newAsyncPool(-1),
		}

		if workflowCfg.On && mongoCfg != nil && mongoCfg.Database != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			connURI := "mongodb://" + mongoCfg.Host + ":" + mongoCfg.Port
			opts := options.Client().
				ApplyURI(connURI).
				SetConnectTimeout(mongoCfg.ConnectTimeOut).
				SetMaxPoolSize(mongoCfg.MaxConnectionPoolSize).
				SetMaxConnIdleTime(mongoCfg.MaxConnectionLifeTime)

			if mongoCfg.UserName != "" && mongoCfg.Password != "" {
				opts.SetAuth(options.Credential{
					AuthSource: mongoCfg.Database,
					Username:   mongoCfg.UserName,
					Password:   mongoCfg.Password,
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
			workflowRecord.mongoCollection = mdb.Database(mongoCfg.Database).Collection(collection.Name)
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
	option    string
}

func (t *TaskStatus) Error() string {
	if t.err == nil {
		return ""
	}
	return t.err.Error()
}

func (t *TaskStatus) Statement() string {
	return string(t.statement)
}
