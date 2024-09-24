package redisx

import (
	_ "embed"
	"github.com/go-redis/redis/v8"
)

var (
	//go:embed lua/hashDuplicate.lua
	hashDuplicateIdLua    string
	HashDuplicateIdScript = redis.NewScript(hashDuplicateIdLua)

	//go:embed lua/addLogicLock.lua
	addLogicLockLua    string
	AddLogicLockScript = redis.NewScript(addLogicLockLua)

	//go:embed lua/saveHSet.lua
	saveHsetLua    string
	SaveHSetScript = redis.NewScript(saveHsetLua)
)
