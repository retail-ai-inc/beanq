// MIT License

// Copyright The RAI Inc.
// The RAI Authors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package base
// @Description:
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

// Retry
//
//	@Description:
//	@param f
//	@param delayTime
//	@return error
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
