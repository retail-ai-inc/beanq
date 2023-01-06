package beanq

import (
	"sync"
	"time"

	"beanq/internal/options"
)

type ConsumerHandler struct {
	Group, Queue string
	ConsumerFun  DoConsumer
}

type Server struct {
	mu    sync.RWMutex
	m     []*ConsumerHandler
	Count int64
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

func NewServer(count int64) *Server {
	if count == 0 {
		count = 10
	}
	return &Server{Count: count}
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
