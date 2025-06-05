package bredis

import (
	_ "embed"

	"github.com/go-redis/redis/v8"
)

var (
	//go:embed scripts/hashDuplicate.lua
	hashDuplicateIdLua    string
	HashDuplicateIdScript = redis.NewScript(hashDuplicateIdLua)

	//go:embed scripts/sequenceByLock.lua
	sequenceByLockLua    string
	SequenceByLockScript = redis.NewScript(sequenceByLockLua)

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
