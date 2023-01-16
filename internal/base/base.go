package base

import (
	"strings"
	"time"

	"beanq/internal/options"
)

func makeKey(group, queue, name string) string {

	if group == "" {
		group = options.DefaultOptions.DefaultGroup
	}
	if queue == "" {
		queue = options.DefaultOptions.DefaultQueueName
	}
	var builder strings.Builder

	builder.WriteString(group)
	builder.WriteString(":")
	builder.WriteString(queue)
	builder.WriteString(":")
	builder.WriteString(name)

	return builder.String()
}
func MakeListKey(group, queue string) string {
	return makeKey(group, queue, "list")
}
func MakeZSetKey(group, queue string) string {
	return makeKey(group, queue, "zset")
}
func MakeStreamKey(group, queue string) string {
	return makeKey(group, queue, "stream")
}

/*
* Retry
*  @Description:

* @param f
* @param delayTime
* @return error
 */
func Retry(f func() error, delayTime time.Duration) error {

	index := 0
	errChan := make(chan error, 1)
	stop := make(chan struct{}, 1)

	go func(timer *time.Timer, err chan error, stop chan struct{}) {
		for {
			select {
			case <-timer.C:
				e := f()
				if e == nil || index >= 2 {
					timer.Stop()
					stop <- struct{}{}
					err <- e
					return
				}
				index++
				timer.Reset(time.Duration(index) * delayTime)
			}
		}
	}(time.NewTimer(time.Duration(index)*delayTime), errChan, stop)

	var e error

	select {
	case <-stop:
		for e = range errChan {
			close(errChan)
			break
		}
	}
	close(stop)
	return e
}
