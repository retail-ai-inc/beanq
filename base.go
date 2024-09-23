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
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"runtime/debug"
	"strings"
	"time"

	"github.com/retail-ai-inc/beanq/helper/stringx"
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
func MakeStreamKey(subType subscribeType, prefix, channel, topic string) string {
	if channel == "" {
		channel = DefaultOptions.DefaultChannel
	}
	if topic == "" {
		topic = DefaultOptions.DefaultTopic
	}
	stream := "normal_stream"
	if subType == sequentialSubscribe {
		stream = "sequential_stream"
	}

	return makeKey(prefix, channel, topic, stream, "stream")
}

// MakeStatusKey create key for type string
func MakeStatusKey(prefix, channel, id string) string {
	return makeKey(prefix, channel, "=-status-=", id)
}

// MakeDynamicKey create key for dynamic
func MakeDynamicKey(prefix, channel string) string {
	if channel == "" {
		channel = DefaultOptions.DefaultChannel
	}

	return makeKey(prefix, channel, "dynamic")
}

// GetChannelAndTopicFromStreamKey get channel and topic
func GetChannelAndTopicFromStreamKey(streamKey string) (channel, topic string) {
	s := strings.SplitN(streamKey, ":", 4)[1:3]
	return s[0], s[1]
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

func MakeTimeUnit(prefix, channel, topic string) string {
	return makeKey(prefix, channel, topic, "time_unit")
}

func MakeFilter(prefix string) string {
	return makeKey(prefix, "filter")
}

func MakeSubKey(prefix, channel, topic string) string {
	return makeKey(prefix, channel, topic, "subKey")
}

const (
	// BeanqLogicGroup it's for beanq-logic-log,multiple consumers can consume those data
	BeanqLogicGroup = "beanq-logic-group"
)

func MakeLogicKey(prefix string) string {
	return makeKey(prefix, "beanq-logic-log")
}

func MakeLogicLock(prefix, id string) string {
	return makeKey(prefix, "beanq-logic-uniqueid", id)
}

func doTimeout(ctx context.Context, f func() error) error {
	errCh := make(chan error, 1)
	go func() {
		defer func() {
			if ne := recover(); ne != nil {
				errCh <- fmt.Errorf("error:%+v,stack:%s", ne, stringx.ByteToString(debug.Stack()))
				return
			}
		}()
		errCh <- f()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// RetryInfo retry=0 means no retries, but it will be executed at least once.
func RetryInfo(ctx context.Context, f func() error, retry int) (i int, err error) {
	for i = 0; i <= retry; i++ {
		err = doTimeout(ctx, f)
		if err == nil {
			return
		}

		waitTime := jitterBackoff(500*time.Millisecond, time.Second, i)
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return i, ctx.Err()
		}
	}
	return
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

func HashKey(id []byte, flake uint64) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(id)
	hashKey := h.Sum64()
	hashKey = hashKey % flake
	return hashKey
}
