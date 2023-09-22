package redisx

import (
	"log"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq"
	"github.com/spf13/viper"
)

var (
	redisOnce sync.Once
	client    *redis.Client
	BqConfig  beanq.BeanqConfig
)

func init() {
	_, f, _, _ := runtime.Caller(0)
	f = path.Dir(f)
	f = filepath.Join(f, "../", "../")

	vp := viper.New()
	vp.AddConfigPath(f)
	vp.SetConfigName("env")
	vp.SetConfigType("json")
	if err := vp.ReadInConfig(); err != nil {
		log.Fatalln(err)
	}
	if err := vp.Unmarshal(&BqConfig); err != nil {
		log.Fatalln(err)
	}
}

func Client() *redis.Client {

	redisOnce.Do(func() {
		client = redis.NewClient(&redis.Options{
			Network:  "",
			Addr:     strings.Join([]string{BqConfig.Redis.Host, BqConfig.Redis.Port}, ":"),
			Username: "",
			Password: BqConfig.Redis.Password,
			DB:       BqConfig.Redis.Database,
		})
	})

	return client
}
