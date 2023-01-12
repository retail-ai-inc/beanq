package beanq

import (
	"log"
	"path/filepath"
	"runtime"
	"time"

	"beanq/helper/logger"
	"github.com/spf13/viper"
)

type BeanqConfig struct {
	Queue struct {
		DebugLog struct {
			On   bool
			Path string
		}
		Redis struct {
			Host               string
			Port               string
			Password           string
			Database           int
			Prefix             string
			Maxretries         int
			PoolSize           int
			MinIdleConnections int
			DialTimeout        time.Duration
			ReadTimeout        time.Duration
			WriteTimeout       time.Duration
			PoolTimeout        time.Duration
		}
		Driver                   string
		JobMaxRetries            int
		KeepJobsInQueue          time.Duration
		KeepFailedJobsInHistory  time.Duration
		KeepSuccessJobsInHistory time.Duration
		MinWorkers               int
	}
}

// This is a global variable to hold the debug logger so that we can log data from anywhere.
var Logger logger.Logger

// Hold the useful configuration settings of beanq so that we can use it quickly from anywhere.
var Config BeanqConfig

func initEnv() {
	var envPath string = "./"
	if _, file, _, ok := runtime.Caller(5); ok {
		envPath = filepath.Dir(file)
	}

	vp := viper.New()
	vp.AddConfigPath(envPath)
	vp.SetConfigType("json")
	vp.SetConfigName("env")

	if err := vp.ReadInConfig(); err != nil {
		log.Fatalf("Unable to open beanq env.json file: %v", err)
	}

	// IMPORTANT: Unmarshal the env.json into global Config object.
	if err := vp.Unmarshal(&Config); err != nil {
		log.Fatalf("Unable to unmarshal the beanq env.json file: %v", err)
	}

}
