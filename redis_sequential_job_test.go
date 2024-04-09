package beanq

import (
	"os"
	"os/signal"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func TestSeq(t *testing.T) {
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        []string{strings.Join([]string{"localhost", "6379"}, ":")},
		Password:     "secret",
		DB:           0,
		MaxRetries:   2,
		DialTimeout:  time.Second * 5,
		ReadTimeout:  time.Second * 4,
		WriteTimeout: time.Second * 3,
		PoolSize:     30,
		MinIdleConns: 10,
		PoolTimeout:  time.Second * 4,
		PoolFIFO:     true,
	})
	seq := newSequentialJob(client)
	go func() {
		seq.watch()
	}()
	for i := 0; i < 5; i++ {
		go func() {
			seq.consume()
		}()
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}
