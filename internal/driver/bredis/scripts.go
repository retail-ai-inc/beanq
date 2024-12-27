package bredis

import (
	_ "embed"
	"github.com/go-redis/redis/v8"
)

var (
	//go:embed scripts/hashDuplicate.lua
	hashDuplicateIdLua    string
	HashDuplicateIdScript = redis.NewScript(hashDuplicateIdLua)

	//go:embed scripts/addLogicLock.lua
	addLogicLockLua    string
	AddLogicLockScript = redis.NewScript(addLogicLockLua)

	//go:embed scripts/saveHSet.lua
	saveHsetLua    string
	SaveHSetScript = redis.NewScript(saveHsetLua)
)
