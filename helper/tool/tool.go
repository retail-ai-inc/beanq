package tool

import (
	"context"
	"fmt"
	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/retail-ai-inc/beanq/v3/internal/boptions"
	"github.com/retail-ai-inc/beanq/v3/internal/btype"
	"github.com/spf13/cast"
	"hash/fnv"
	"math"
	"math/rand"
	"runtime/debug"
	"strings"
	"time"
)

func makeKey(keys ...string) string {

	return strings.Join(keys, ":")

}

// MakeZSetKey create redis key for type sorted set
func MakeZSetKey(prefix, channel, topic string) string {
	if channel == "" {
		channel = boptions.DefaultOptions.DefaultChannel
	}
	if topic == "" {
		topic = boptions.DefaultOptions.DefaultTopic
	}
	channel = strings.Join([]string{"{", channel}, "")
	topic = strings.Join([]string{topic, "}"}, "")
	return makeKey(prefix, channel, topic, "zset")
}

// MakeStreamKey create key for type stream
func MakeStreamKey(subType btype.SubscribeType, prefix, channel, topic string) string {

	if channel == "" {
		channel = boptions.DefaultOptions.DefaultChannel
	}
	if topic == "" {
		topic = boptions.DefaultOptions.DefaultTopic
	}
	stream := "normal_stream"
	if subType == btype.SequentialSubscribe {
		stream = "sequential_stream"
	}
	if subType == btype.DelaySubscribe {
		stream = "delay_stream"
	}
	channel = strings.Join([]string{"{", channel}, "")
	topic = strings.Join([]string{topic, "}"}, "")
	return makeKey(prefix, channel, topic, stream, "stream")
}

// MakeStatusKey create key for type string
func MakeStatusKey(prefix, channel, topic, id string) string {

	channel = strings.Join([]string{"{", channel}, "")
	topic = strings.Join([]string{topic, "}"}, "")

	return makeKey(prefix, channel, topic, "=-status-=", id)
}

// MakeDynamicKey create key for dynamic
func MakeDynamicKey(prefix, channel string) string {
	if channel == "" {
		channel = boptions.DefaultOptions.DefaultChannel
	}

	return makeKey(prefix, channel, "dynamic")
}

// GetChannelAndTopicFromStreamKey get channel and topic
func GetChannelAndTopicFromStreamKey(streamKey string) (channel, topic string) {
	s := strings.SplitN(streamKey, ":", 4)[1:3]
	return s[0], s[1]
}

const (
	BeanqHostName = "beanq-host-name"
)

const (
	// BeanqLogGroup it's for beanq-logic-log,multiple consumers can consume those data
	BeanqLogGroup = "beanq-log-group"
)

func MakeLogicKey(prefix string) string {
	return makeKey(prefix, "beanq-logic-log")
}

func doTimeout(ctx context.Context, f func() error) error {
	errCh := make(chan error, 1)
	go func() {
		defer func() {
			if ne := recover(); ne != nil {
				errCh <- fmt.Errorf("error:%+v,stack:%s", ne, string(debug.Stack()))
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

		waitTime := JitterBackoff(500*time.Millisecond, time.Second, i)
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return i, ctx.Err()
		}
	}
	return
}

func JitterBackoff(min, max time.Duration, attempt int) time.Duration {
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

func JsonDecode[T map[string]any | map[string]string](data string, m *T) error {

	if err := json.NewDecoder(strings.NewReader(data)).Decode(m); err != nil {
		return err
	}
	return nil
}

// Commands
// sort in reverse order based on the `usec_per_call` field
type Commands []map[string]any

func (t Commands) Len() int {
	return len(t)
}

func (t Commands) Less(i, j int) bool {
	return cast.ToFloat64(t[j]["usec_per_call"]) < cast.ToFloat64(t[i]["usec_per_call"])
}

func (t Commands) Swap(i, j int) {
	t[j], t[i] = t[i], t[j]
}
