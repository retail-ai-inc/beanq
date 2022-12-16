package server

import (
	"beanq/task"
	"sync"
)

type ConsumerHandler struct {
	Group, Queue string
	ConsumerFun  task.DoConsumer
}
type Server struct {
	mu    sync.RWMutex
	m     []*ConsumerHandler
	Count int64
}

func NewServer(count int64) *Server {
	if count == 0 {
		count = 10
	}
	return &Server{Count: count}
}
func (t *Server) Register(group, queue string, consumerFun task.DoConsumer) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if group == "" {
		group = task.DefaultOptions.DefaultGroup
	}
	if queue == "" {
		queue = task.DefaultOptions.DefaultQueueName
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
