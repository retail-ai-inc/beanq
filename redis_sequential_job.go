package beanq

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/helper/redisx"
	"golang.org/x/net/context"
)

type sequentialJob struct {
	cond   *sync.Cond
	client redis.UniversalClient

	streamStr string
	ISequential
	data chan redis.XStream
}

var (
	status atomic.Uint32
)

func newSequentialJob(client redis.UniversalClient) *sequentialJob {
	return &sequentialJob{
		cond:        sync.NewCond(new(sync.Mutex)),
		client:      client,
		streamStr:   "",
		ISequential: newSequential(),
		data:        make(chan redis.XStream, 1),
	}
}

func (t *sequentialJob) consume() {
	for {
		data := <-t.data
		fmt.Println(data)
		time.Sleep(3 * time.Second)
		status.Store(0)
	}
}

func (t *sequentialJob) watch() {
	consumer := "test_queue:delay-channel:delay-topic:stream"
	readGroupArgs := redisx.NewReadGroupArgs("delay-channel", consumer, []string{consumer, ">"}, 1, 10*time.Second)

	for {
		if status.Load() == 1 {
			continue
		}
		cmd := t.client.XReadGroup(context.Background(), readGroupArgs)
		vals := cmd.Val()
		if len(vals) <= 0 {
			continue
		}
		t.data <- vals[0]
		status.Store(1)
	}

}
