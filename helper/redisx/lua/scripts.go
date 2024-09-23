package lua

import (
	_ "embed"
	"github.com/go-redis/redis/v8"
)

var (
	//go:embed hashDuplicate.lua
	hashDuplicateIdLua    string
	HashDuplicateIdScript = redis.NewScript(hashDuplicateIdLua)

	//go:embed addLogicLock.lua
	addLogicLockLua    string
	AddLogicLockScript = redis.NewScript(addLogicLockLua)

	//go:embed saveHSet.lua
	saveHsetLua    string
	SaveHSetScript = redis.NewScript(saveHsetLua)
)
