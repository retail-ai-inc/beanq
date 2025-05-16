package beanq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/internal/driver/bredis"
	"github.com/spf13/cast"
)

const (
	// StatusPrepared status for global trans status.
	StatusPrepared = "prepared"
	// StatusSucceed status for global/branch trans status.
	StatusSucceed = "succeed"
	// StatusFailed status for global/branch trans status.
	// NOTE: change global status to failed can stop trigger (Not recommended in production env)
	StatusFailed = "failed"
	// StatusAborting status for global trans status.
	StatusAborting = "aborting"
)

var (
	ErrNotFound       = errors.New("storage: NotFound")
	ErrUniqueConflict = errors.New("storage: UniqueKeyConflict")
	ErrUndefined      = errors.New("storage: Undefined")
)

type TransStore interface {
	FindGlobal(ctx context.Context, gid string) (*TransGlobal, error)
	ScanGlobals(ctx context.Context, position *string, limit int64, condition TransGlobalScanCondition) ([]TransGlobal, error)
	FindBranches(ctx context.Context, gid string) ([]TransBranch, error)
	MaySaveNew(ctx context.Context, global *TransGlobal, branches []TransBranch) error
	LockGlobalSaveBranches(ctx context.Context, gid string, status string, branches []TransBranch, branchStart int) error
	ChangeGlobalStatus(ctx context.Context, global *TransGlobal, newStatus string, updates []string, finished bool, finishedExpire time.Duration) error
}

type transStore struct {
	prefix string
	expire time.Duration
	client redis.UniversalClient
}

var _ TransStore = (*transStore)(nil)

type TransGlobalScanCondition struct {
	Status          string
	TransType       string
	CreateTimeStart time.Time
	CreateTimeEnd   time.Time
}

type TransGlobal struct {
	Gid              string              `json:"gid,omitempty"`
	TransType        string              `json:"trans_type,omitempty"`
	Steps            []map[string]string `json:"steps,omitempty" gorm:"-"`
	Payloads         []string            `json:"payloads,omitempty" gorm:"-"`
	Status           string              `json:"status,omitempty"`
	QueryPrepared    string              `json:"query_prepared,omitempty"`
	FinishTime       *time.Time          `json:"finish_time,omitempty"`
	RollbackTime     *time.Time          `json:"rollback_time,omitempty"`
	Reason           string              `json:"reason,omitempty"`
	Options          string              `json:"options,omitempty"`
	CustomData       string              `json:"custom_data,omitempty"`
	NextCronInterval time.Duration       `json:"next_cron_interval,omitempty"`
	NextCronTime     *time.Time          `json:"next_cron_time,omitempty"`
	Owner            string              `json:"owner,omitempty"`
	CreateTime       *time.Time          `json:"create_time"`
	UpdateTime       *time.Time          `json:"update_time"`
	MessageData      string              `json:"message_data,omitempty"`
	Message          *Message            `json:"-"`
}

type TransBranch struct {
	Index        int        `json:"index,omitempty"`
	Gid          string     `json:"gid,omitempty"`
	TaskID       string     `json:"task_id,omitempty"`
	Statement    string     `json:"url,omitempty"`
	BinData      []byte     `json:"bin_data,omitempty"`
	BranchID     string     `json:"branch_id,omitempty"`
	Op           string     `json:"op,omitempty"`
	Status       string     `json:"status,omitempty"`
	FinishTime   *time.Time `json:"finish_time,omitempty"`
	RollbackTime *time.Time `json:"rollback_time,omitempty"`
	Error        error      `json:"-"`
	CreateTime   *time.Time `json:"create_time"`
	UpdateTime   *time.Time `json:"update_time"`
}

func NewTransStore(client redis.UniversalClient, prefix string, dataExpire time.Duration) *transStore {
	return &transStore{
		prefix: prefix,
		client: client,
		expire: dataExpire,
	}
}

func (t *transStore) FindGlobal(ctx context.Context, gid string) (*TransGlobal, error) {
	r, err := t.client.Get(ctx, t.prefix+"_g_"+gid).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("[transStore.FindGlobal] client get failed: %w", err)
	}

	var transGlobal TransGlobal
	err = json.Unmarshal([]byte(r), &transGlobal)
	if err != nil {
		return nil, fmt.Errorf("[transStore.FindGlobal] decode failed: %w", err)
	}

	return &transGlobal, nil
}

func (t *transStore) ScanGlobals(ctx context.Context, position *string, limit int64, condition TransGlobalScanCondition) ([]TransGlobal, error) {
	var positionIndex uint64
	if position != nil && *position != "" {
		var err error
		positionIndex, err = cast.ToUint64E(*position)
		if err != nil {
			return nil, err
		}
	}

	var globals []TransGlobal

	for {
		limit -= int64(len(globals))
		keys, nextCursor, err := t.client.Scan(ctx, positionIndex, t.prefix+"_g_*", limit).Result()
		if err != nil {
			return nil, err
		}

		if len(keys) > 0 {
			values, err := t.client.MGet(ctx, keys...).Result()
			if err != nil {
				return nil, err
			}
			for _, v := range values {
				var global TransGlobal
				err = json.Unmarshal([]byte(v.(string)), &global)
				if err != nil {
					return nil, err
				}

				if (condition.Status == "" || global.Status == condition.Status) &&
					(condition.TransType == "" || global.TransType == condition.TransType) &&
					(condition.CreateTimeStart.IsZero() || global.CreateTime.After(condition.CreateTimeStart)) &&
					(condition.CreateTimeEnd.IsZero() || global.CreateTime.Before(condition.CreateTimeEnd)) {
					globals = append(globals, global)
				}

				// redis.Scan may return more records than limit
				if len(globals) >= int(limit) {
					break
				}
			}
		}

		positionIndex = nextCursor
		if len(globals) >= int(limit) || nextCursor == 0 {
			break
		}
	}

	if positionIndex > 0 {
		*position = fmt.Sprintf("%d", positionIndex)
	} else {
		*position = ""
	}

	return globals, nil
}

func (t *transStore) FindBranches(ctx context.Context, gid string) ([]TransBranch, error) {
	rs, err := t.client.LRange(ctx, t.prefix+"_b_"+gid, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("[FindBranches] LRange failed: %w", err)
	}

	branches := make([]TransBranch, len(rs))
	for k, v := range rs {
		var branch TransBranch
		err := json.Unmarshal([]byte(v), &branch)
		if err != nil {
			return nil, fmt.Errorf("[FindBranches] decode failed: %w", err)
		}
		branches[k] = branch
	}

	return branches, nil
}

func (t *transStore) MaySaveNew(ctx context.Context, global *TransGlobal, branches []TransBranch) error {
	args, err := newArgList(t.prefix, t.expire).
		AppendGid(global.Gid).
		AppendObject(global).
		AppendRaw(global.NextCronTime.Unix()).
		AppendRaw(global.Gid).
		AppendRaw(global.Status).
		AppendBranches(branches).
		Result()
	if err != nil {
		return fmt.Errorf("[MaySaveNew] newArgList failed: %w", err)
	}

	global.Steps = nil
	global.Payloads = nil

	ret, err := bredis.SaveNewTransScript.Run(ctx, t.client, args.Keys, args.List...).Result()

	return handleRedisResult(ret, err)
}

func (t *transStore) LockGlobalSaveBranches(ctx context.Context, gid string, status string, branches []TransBranch, branchStart int) error {
	args, err := newArgList(t.prefix, t.expire).
		AppendGid(gid).
		AppendRaw(status).
		AppendRaw(branchStart).
		AppendBranches(branches).
		Result()
	if err != nil {
		return fmt.Errorf("[LockGlobalSaveBranches] newArgList failed: %w", err)
	}

	ret, err := bredis.SaveBranchesScript.Run(ctx, t.client, args.Keys, args.List...).Result()
	return handleRedisResult(ret, err)
}

func (t *transStore) ChangeGlobalStatus(
	ctx context.Context,
	global *TransGlobal,
	newStatus string,
	updates []string,
	finished bool,
	finishedExpire time.Duration,
) error {
	if finishedExpire < 0 {
		// finished Trans data will expire in 1 days as default.
		finishedExpire = time.Hour * 24
	}

	oldStatus := global.Status
	global.Status = newStatus

	args, err := newArgList(t.prefix, t.expire).
		AppendGid(global.Gid).
		AppendObject(global).
		AppendRaw(oldStatus).
		AppendRaw(finished).
		AppendRaw(global.Gid).
		AppendRaw(newStatus).
		AppendObject(finishedExpire.Seconds()).
		Result()
	if err != nil {
		return fmt.Errorf("[ChangeGlobalStatus] newArgList failed: %w", err)
	}

	ret, err := bredis.ChangeGlobalStatusScript.Run(ctx, t.client, args.Keys, args.List...).Result()

	return handleRedisResult(ret, err)
}

func handleRedisResult(ret interface{}, err error) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, redis.Nil) {
		return err
	}

	s, _ := ret.(string)
	err, ok := map[string]error{
		"NOT_FOUND":       ErrNotFound,
		"UNIQUE_CONFLICT": ErrUniqueConflict,
	}[s]

	if !ok {
		return ErrUndefined
	}
	return err
}

type argList struct {
	errs   error
	prefix string
	Keys   []string      // 1 global trans, 2 branches, 3 indices, 4 status
	List   []interface{} // 1 redis prefix, 2 data expire
}

func newArgList(prefix string, expire time.Duration) *argList {
	a := &argList{
		prefix: prefix,
	}

	return a.AppendRaw(prefix).AppendObject(expire.Seconds())
}

func (a *argList) Result() (*argList, error) {
	return a, a.errs
}

func (a *argList) AppendGid(gid string) *argList {
	if a.errs != nil {
		return a
	}

	a.Keys = append(a.Keys, a.prefix+"_g_"+gid)
	a.Keys = append(a.Keys, a.prefix+"_b_"+gid)
	a.Keys = append(a.Keys, a.prefix+"_u")
	a.Keys = append(a.Keys, a.prefix+"_s_"+gid)
	return a
}

func (a *argList) AppendRaw(v interface{}) *argList {
	if a.errs != nil {
		return a
	}

	a.List = append(a.List, v)
	return a
}

func (a *argList) AppendObject(v interface{}) *argList {
	if a.errs != nil {
		return a
	}

	bs, err := json.Marshal(v)
	if err != nil {
		a.errs = errors.Join(a.errs, err)
		return a
	}

	return a.AppendRaw(string(bs))
}

func (a *argList) AppendBranches(branches []TransBranch) *argList {
	if a.errs != nil {
		return a
	}

	for _, b := range branches {
		bs, err := json.Marshal(b)
		if err != nil {
			a.errs = errors.Join(a.errs, err)
			return a
		}
		a.AppendRaw(string(bs))
	}

	return a
}

func NewTransGlobal(message *Message) (*TransGlobal, error) {
	t := &TransGlobal{
		Message: message,
	}

	t.Gid = t.Message.Id
	t.TransType = "workflow"
	t.Status = StatusPrepared

	t.NextCronInterval++
	nextCronTime := time.Now().Add(jitterBackoff(t.NextCronInterval))
	t.NextCronTime = &nextCronTime

	bs, err := json.Marshal(t.Message)
	if err != nil {
		return nil, fmt.Errorf("SaveNew failed: %w", err)
	}

	t.MessageData = string(bs)
	if t.MessageData == "{}" {
		t.MessageData = ""
	}

	now := time.Now()
	t.CreateTime = &now
	t.UpdateTime = &now

	return t, nil
}

var (
	minWaitTime = time.Duration(100) * time.Millisecond
	maxWaitTime = time.Duration(2000) * time.Millisecond
)

func jitterBackoff(attempt time.Duration) time.Duration {
	base := float64(minWaitTime)
	capLevel := float64(maxWaitTime)

	temp := math.Min(capLevel, base*math.Exp2(float64(attempt)))
	ri := time.Duration(temp / 2)
	result := randDuration(ri)

	if result < minWaitTime {
		result = minWaitTime
	}

	return result
}

func randDuration(center time.Duration) time.Duration {
	ri := int64(center)
	jitter := rand.Int63n(ri)
	return time.Duration(math.Abs(float64(ri + jitter)))
}
