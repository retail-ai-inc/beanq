package bredis

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	ScriptHashDuplicate          = "hashDuplicate"
	ScriptSequenceByLock         = "sequenceByLock"
	ScriptSequenceQueueMeta      = "sequenceQueueMeta"
	ScriptSequenceQueueEnqueue   = "sequenceQueueEnqueue"
	ScriptSequenceQueueAcquire   = "sequenceQueueAcquire"
	ScriptSequenceQueueHeartbeat = "sequenceQueueHeartbeat"
	ScriptSequenceQueueFinalize  = "sequenceQueueFinalize"
	ScriptAddLogicLock           = "addLogicLock"
	ScriptSaveHSet               = "saveHSet"
	ScriptSaveNewTrans           = "saveNewTrans"
	ScriptSaveBranches           = "saveBranches"
	ScriptChangeGlobalStatus     = "changeGlobalStatus"
)

var (
	//go:embed scripts/hashDuplicate.lua
	hashDuplicateIdLua    string
	HashDuplicateIdScript = redis.NewScript(hashDuplicateIdLua)

	//go:embed scripts/sequenceByLock.lua
	sequenceByLockLua    string
	SequenceByLockScript = redis.NewScript(sequenceByLockLua)

	//go:embed scripts/sequenceQueueMeta.lua
	sequenceQueueMetaLua    string
	SequenceQueueMetaScript = redis.NewScript(sequenceQueueMetaLua)

	//go:embed scripts/sequenceQueueEnqueue.lua
	sequenceQueueEnqueueLua    string
	SequenceQueueEnqueueScript = redis.NewScript(sequenceQueueEnqueueLua)

	//go:embed scripts/sequenceQueueAcquire.lua
	sequenceQueueAcquireLua    string
	SequenceQueueAcquireScript = redis.NewScript(sequenceQueueAcquireLua)

	//go:embed scripts/sequenceQueueHeartbeat.lua
	sequenceQueueHeartbeatLua    string
	SequenceQueueHeartbeatScript = redis.NewScript(sequenceQueueHeartbeatLua)

	//go:embed scripts/sequenceQueueFinalize.lua
	sequenceQueueFinalizeLua    string
	SequenceQueueFinalizeScript = redis.NewScript(sequenceQueueFinalizeLua)

	//go:embed scripts/addLogicLock.lua
	addLogicLockLua    string
	AddLogicLockScript = redis.NewScript(addLogicLockLua)

	//go:embed scripts/saveHSet.lua
	saveHsetLua    string
	SaveHSetScript = redis.NewScript(saveHsetLua)

	//go:embed scripts/saveNewTrans.lua
	saveNewTransLua    string
	SaveNewTransScript = redis.NewScript(saveNewTransLua)

	//go:embed scripts/saveBranches.lua
	saveBranchesLua    string
	SaveBranchesScript = redis.NewScript(saveBranchesLua)

	//go:embed scripts/changeGlobalStatus.lua
	changeGlobalStatusLua    string
	ChangeGlobalStatusScript = redis.NewScript(changeGlobalStatusLua)
)

// ScriptCatalog gives Redis Lua scripts a single named access point.
type ScriptCatalog struct {
	scripts map[string]*redis.Script
}

func NewScriptCatalog(scripts map[string]*redis.Script) *ScriptCatalog {
	catalog := &ScriptCatalog{scripts: make(map[string]*redis.Script, len(scripts))}
	for name, script := range scripts {
		catalog.scripts[name] = script
	}
	return catalog
}

func DefaultScriptCatalog() *ScriptCatalog {
	return NewScriptCatalog(map[string]*redis.Script{
		ScriptHashDuplicate:          HashDuplicateIdScript,
		ScriptSequenceByLock:         SequenceByLockScript,
		ScriptSequenceQueueMeta:      SequenceQueueMetaScript,
		ScriptSequenceQueueEnqueue:   SequenceQueueEnqueueScript,
		ScriptSequenceQueueAcquire:   SequenceQueueAcquireScript,
		ScriptSequenceQueueHeartbeat: SequenceQueueHeartbeatScript,
		ScriptSequenceQueueFinalize:  SequenceQueueFinalizeScript,
		ScriptAddLogicLock:           AddLogicLockScript,
		ScriptSaveHSet:               SaveHSetScript,
		ScriptSaveNewTrans:           SaveNewTransScript,
		ScriptSaveBranches:           SaveBranchesScript,
		ScriptChangeGlobalStatus:     ChangeGlobalStatusScript,
	})
}

func (c *ScriptCatalog) Run(ctx context.Context, client redis.Scripter, name string, keys []string, args ...any) (any, error) {
	if c == nil {
		return nil, fmt.Errorf("redis script catalog is nil")
	}
	script, ok := c.scripts[name]
	if !ok {
		return nil, fmt.Errorf("redis script %q is not registered", name)
	}
	return script.Run(ctx, client, keys, args...).Result()
}
