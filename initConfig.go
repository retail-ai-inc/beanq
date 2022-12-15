package beanq

import (
	"github.com/spf13/viper"
	"log"
	"path/filepath"
	"runtime"
	"time"
)

type EnvJson struct {
	Version     string `json:"version"`
	ProjectName string `json:"projectName"`
	Environment string `json:"environment"`
	Queue       queues `json:"queue"`
}
type queues struct {
	Broker                   string        `json:"broker"`
	JobMaxRetries            uint64        `json:"jobMaxRetries"`
	KeepJobsInQueue          time.Duration `json:"keepJobsInQueue"`
	KeepFailedJobsInHistory  time.Duration `json:"keepFailedJobsInHistory"`
	KeepSuccessJobsInHistory time.Duration `json:"keepSuccessJobsInHistory"`
	MinWorkers               uint64        `json:"minWorkers"`
	Redis                    redisq        `json:"redis"`
}
type redisq struct {
	Host               string        `json:"host"`
	Port               uint64        `json:"port"`
	Password           string        `json:"password"`
	Name               string        `json:"name"`
	Db                 int64         `json:"db"`
	Prefix             string        `json:"prefix"`
	MaxRetries         int64         `json:"maxRetries"`
	PoolSize           uint64        `json:"poolSize"`
	MinIdleConnections uint64        `json:"minIdleConnections"`
	DialTimeout        time.Duration `json:"dialTimeout"`
	ReadTimeout        time.Duration `json:"readTimeout"`
	WriteTimeout       time.Duration `json:"writeTimeout"`
	PoolTimeout        time.Duration `json:"poolTimeout"`
}

var Env = EnvJson{}

func InitJson() {
	var envPath string
	if _, file, _, ok := runtime.Caller(1); ok {
		envPath = filepath.Dir(file)
	}
	if envPath == "" {
		log.Fatal("config directory is empty")
	}
	vp := viper.New()
	vp.SetConfigName("env")
	vp.SetConfigType("json")
	vp.AddConfigPath(envPath)
	if err := vp.ReadInConfig(); err != nil {
		log.Fatalf("ConfigError:%s \n", err.Error())
	}
	if err := vp.Unmarshal(&Env); err != nil {
		log.Fatalf("DataError:%s \n", err.Error())
	}
}
