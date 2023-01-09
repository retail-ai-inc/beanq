package beanq

import (
	"sync"
	"time"

	"beanq/helper/file"
	"beanq/internal/options"

	"github.com/labstack/gommon/log"
	"github.com/spf13/viper"
)

type ConsumerHandler struct {
	Group, Queue string
	ConsumerFun  DoConsumer
}

type Server struct {
	mu    sync.RWMutex
	m     []*ConsumerHandler
	Count int
}

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

func NewServer() *Server {
	viper.AddConfigPath(".")
	viper.SetConfigType("json")
	viper.SetConfigName("env")

	// Initialize the beanq consumer log
	Logger = log.New("beanq")

	if err := viper.ReadInConfig(); err != nil {
		Logger.Fatalf("Unable to open env.json file: %v Beanq 🚀 crash landed. Exiting...\n", err)
	}

	// IMPORTANT: Unmarshal the env.json into global Config object.
	if err := viper.Unmarshal(&Config); err != nil {
		Logger.Fatalf("Unable to unmarshal the env.json file: %v Beanq 🚀 crash landed. Exiting...\n", err)
	}

	// IMPORTANT: Configure debug log. If `path` is empty then push the log into `stdout`.
	if Config.Queue.DebugLog.Path != "" {
		if file, err := file.OpenFile(Config.Queue.DebugLog.Path); err != nil {
			Logger.Fatalf("Unable to open log file: %v Server 🚀  crash landed. Exiting...\n", err)
		} else {
			Logger.SetOutput(file)
		}
	}

	// Set the default log level as DEBUG.
	Logger.SetLevel(log.DEBUG)

	if Config.Queue.MinWorkers == 0 {
		Config.Queue.MinWorkers = options.DefaultOptions.MinWorkers
	}

	return &Server{Count: Config.Queue.MinWorkers}
}

func (t *Server) Register(group, queue string, consumerFun DoConsumer) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if group == "" {
		group = options.DefaultOptions.DefaultGroup
	}
	if queue == "" {
		queue = options.DefaultOptions.DefaultQueueName
	}

	t.m = append(t.m, &ConsumerHandler{
		Group:       group,
		Queue:       queue,
		ConsumerFun: consumerFun,
	})
}

func (t *Server) Consumers() []*ConsumerHandler {
	return t.m
}
