package redisx

import (
	"github.com/redis/go-redis/v9"
)

func RClient(addr, password string, db int) *redis.Client {

	client := redis.NewClient(&redis.Options{
		Network:  "",
		Addr:     addr,
		Username: "",
		Password: password,
		DB:       db,
	})

	return client
}
