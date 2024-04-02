package beanq

import (
	"sync"

	"github.com/go-redis/redis/v8"
)

type sequentialJob struct {
	cond      *sync.Cond
	client    redis.UniversalClient
	stream    chan redis.XStream
	streamStr string
	ISequential
}

var (
	status uint32
)

func newSequentialJob(client redis.UniversalClient) *sequentialJob {
	return &sequentialJob{
		cond:        sync.NewCond(new(sync.Mutex)),
		client:      client,
		stream:      make(chan redis.XStream, 1),
		streamStr:   "",
		ISequential: newSequential(),
	}
}
