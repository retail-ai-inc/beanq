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

package beanq

import (
	"math"
	"math/rand"
	"strings"
	"time"
)

func makeKey(keys ...string) string {

	return strings.Join(keys, ":")

}

// MakeListKey create redis key for type :list
func MakeListKey(prefix, channel, topic string) string {
	if channel == "" {
		channel = DefaultOptions.DefaultChannel
	}
	if topic == "" {
		topic = DefaultOptions.DefaultTopic
	}
	return makeKey(prefix, channel, topic, "list")
}

// MakeZSetKey create redis key for type sorted set
func MakeZSetKey(prefix, channel, topic string) string {
	if channel == "" {
		channel = DefaultOptions.DefaultChannel
	}
	if topic == "" {
		topic = DefaultOptions.DefaultTopic
	}
	return makeKey(prefix, channel, topic, "zset")
}

// MakeStreamKey create key for type stream
func MakeStreamKey(prefix, channel, topic string) string {
	if channel == "" {
		channel = DefaultOptions.DefaultChannel
	}
	if topic == "" {
		topic = DefaultOptions.DefaultTopic
	}
	return makeKey(prefix, channel, topic, "stream")
}

// MakeDeadLetterStreamKey create key for type stream,mainly dead letter
func MakeDeadLetterStreamKey(prefix, channel, topic string) string {
	if channel == "" {
		channel = DefaultOptions.DefaultChannel
	}
	if topic == "" {
		topic = DefaultOptions.DefaultTopic
	}
	return makeKey(prefix, channel, topic, "dead_letter_stream")
}

func MakeLogKey(prefix, resultType string) string {
	return makeKey(prefix, "logs", resultType)
}

func MakeHealthKey(prefix string) string {
	return makeKey(prefix, "health_checker")
}
func MakeTimeUnit(prefix, channel, topic string) string {
	return makeKey(prefix, channel, topic, "time_unit")
}

func RetryInfo(f func() error, retry int) error {
	index := 0
	errChan := make(chan error, 1)
	stop := make(chan struct{}, 1)

	go func(timer *time.Timer, err chan error, stop chan struct{}) {
		for {
			select {
			case <-timer.C:
				e := f()
				if e == nil || index >= retry {
					timer.Stop()
					stop <- struct{}{}
					err <- e
					return
				}
				index++
				timer.Reset(jitterBackoff(500*time.Millisecond, time.Second, retry))
			}
		}
	}(time.NewTimer(time.Duration(index)*time.Millisecond), errChan, stop)

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
func jitterBackoff(min, max time.Duration, attempt int) time.Duration {
	base := float64(min)
	capLevel := float64(max)

	temp := math.Min(capLevel, base*math.Exp2(float64(attempt)))
	ri := time.Duration(temp / 2)
	dura := randDuration(ri)

	if dura < min {
		dura = min
	}

	return dura
}

func randDuration(center time.Duration) time.Duration {
	var ri = int64(center)
	var jitter = rand.Int63n(ri)
	return time.Duration(math.Abs(float64(ri + jitter)))
}
