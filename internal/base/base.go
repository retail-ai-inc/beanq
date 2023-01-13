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

	retryFlag := make(chan error)
	stopRetry := make(chan bool, 1)

	go func(duration time.Duration, errChan chan error, stop chan bool) {

		var index time.Duration = 0
		var retryCount time.Duration = 2

		for {
			go time.AfterFunc(index*duration, func() {
				errChan <- f()
			})

			err := <-errChan
			if err == nil {
				stop <- true
				close(errChan)
				break
			}
			if index == retryCount {
				stop <- true
				errChan <- err
				break
			}
			index++
		}
	}(delayTime, retryFlag, stopRetry)

	var err error
	select {
	case <-stopRetry:
		for v := range retryFlag {
			err = v
			if v != nil {
				err = v
				close(retryFlag)
				break
			}
		}
	}
	close(stopRetry)
	return err
}
