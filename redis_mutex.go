// redis_mutex.go

// This file includes modified code from redsync by original author.
// Original source: https://github.com/go-redsync/redsync
// License: BSD 3-Clause License

package beanq

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	mRand "math/rand"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hashicorp/go-multierror"
)

// ErrFailed is the error resulting if Redsync fails to acquire the lock after exhausting all retries.
var ErrFailed = errors.New("failed to acquire lock")

// ErrExtendFailed is the error resulting if Redsync fails to extend the lock.
var ErrExtendFailed = errors.New("failed to extend lock")

// ErrLockAlreadyExpired is the error resulting if trying to unlock the lock which already expired.
var ErrLockAlreadyExpired = errors.New("failed to unlock, lock was already expired")

// A DelayFunc is used to decide the amount of time to wait between retries.
type DelayFunc func(tries int) time.Duration

type MuxClient struct {
	expireTime time.Duration
	prefix     string
	client     redis.UniversalClient
}

func NewMuxClient(client redis.UniversalClient) *MuxClient {
	return &MuxClient{
		expireTime: time.Second * 8,
		prefix:     "mutex",
		client:     client,
	}
}

func (p *MuxClient) SetPrefix(prefix string) *MuxClient {
	p.prefix = prefix
	return p
}

func (p *MuxClient) SetExpireTime(expireTime time.Duration) *MuxClient {
	p.expireTime = expireTime
	return p
}

func (p *MuxClient) NewMutex(name string, options ...MuxOption) *Mutex {
	pools := []redis.UniversalClient{
		p.client,
	}
	m := &Mutex{
		name:   strings.Join([]string{p.prefix, name}, ":"),
		expiry: p.expireTime,
		tries:  32,
		delayFunc: func(tries int) time.Duration {
			return time.Duration(mRand.Intn(maxRetryDelayMilliSec-minRetryDelayMilliSec)+minRetryDelayMilliSec) * time.Millisecond
		},
		genValueFunc:  genValue,
		driftFactor:   0.01,
		timeoutFactor: 0.1,
		quorum:        len(pools)/2 + 1,
		pools:         pools,
	}

	for _, o := range options {
		o.Apply(m)
	}

	if m.shuffle {
		mRand.Shuffle(len(pools), func(i, j int) {
			pools[i], pools[j] = pools[j], pools[i]
		})
	}
	return m
}

// A Mutex is a distributed mutual exclusion lock.
type Mutex struct {
	name   string
	expiry time.Duration

	tries     int
	delayFunc DelayFunc

	driftFactor   float64
	timeoutFactor float64

	quorum int

	genValueFunc  func() (string, error)
	value         string
	until         time.Time
	shuffle       bool
	failFast      bool
	setNXOnExtend bool

	pools []redis.UniversalClient
}

// Name returns mutex name (i.e. the Redis key).
func (m *Mutex) Name() string {
	return m.name
}

// Value returns the current random value. The value will be empty until a lock is acquired (or WithValue option is used).
func (m *Mutex) Value() string {
	return m.value
}

// Until returns the time of validity of acquired lock. The value will be zero value until a lock is acquired.
func (m *Mutex) Until() time.Time {
	return m.until
}

// LockContext locks m. In case it returns an error on failure, you may retry to acquire the lock by calling this method again.
func (m *Mutex) LockContext(ctx context.Context) error {
	return m.lockContext(ctx, m.tries)
}

// lockContext locks m. In case it returns an error on failure, you may retry to acquire the lock by calling this method again.
func (m *Mutex) lockContext(ctx context.Context, tries int) error {
	if ctx == nil {
		ctx = context.Background()
	}

	value, err := m.genValueFunc()
	if err != nil {
		return err
	}

	var timer *time.Timer
	for i := 0; i < tries; i++ {
		if i != 0 {
			if timer == nil {
				timer = time.NewTimer(m.delayFunc(i))
			} else {
				timer.Reset(m.delayFunc(i))
			}

			select {
			case <-ctx.Done():
				timer.Stop()
				// Exit early if the context is done.
				return ErrFailed
			case <-timer.C:
				// Fall-through when the delay timer completes.
			}
		}

		start := time.Now()

		n, err := func() (int, error) {
			ctx, cancel := context.WithTimeout(ctx, time.Duration(int64(float64(m.expiry)*m.timeoutFactor)))
			defer cancel()
			return m.actOnPoolsAsync(func(client redis.UniversalClient) (bool, error) {
				return m.acquire(ctx, client, value)
			})
		}()

		now := time.Now()
		until := now.Add(m.expiry - now.Sub(start) - time.Duration(int64(float64(m.expiry)*m.driftFactor)))

		if n >= m.quorum && now.Before(until) {
			m.value = value
			m.until = until
			return nil
		}
		_, _ = func() (int, error) {
			ctx, cancel := context.WithTimeout(ctx, time.Duration(int64(float64(m.expiry)*m.timeoutFactor)))
			defer cancel()
			return m.actOnPoolsAsync(func(client redis.UniversalClient) (bool, error) {
				return m.release(ctx, client, value)
			})
		}()
		if i == tries-1 && err != nil {
			return err
		}
	}

	return ErrFailed
}

// UnlockContext unlocks m and returns the status of unlock.
func (m *Mutex) UnlockContext(ctx context.Context) (bool, error) {
	n, err := m.actOnPoolsAsync(func(client redis.UniversalClient) (bool, error) {
		return m.release(ctx, client, m.value)
	})
	if n < m.quorum {
		return false, err
	}
	return true, nil
}

// ExtendContext resets the mutex's expiry and returns the status of expiry extension.
func (m *Mutex) ExtendContext(ctx context.Context) (bool, error) {
	start := time.Now()
	n, err := m.actOnPoolsAsync(func(client redis.UniversalClient) (bool, error) {
		return m.touch(ctx, client, m.value, int(m.expiry/time.Millisecond))
	})
	if n < m.quorum {
		return false, err
	}
	now := time.Now()
	until := now.Add(m.expiry - now.Sub(start) - time.Duration(int64(float64(m.expiry)*m.driftFactor)))
	if now.Before(until) {
		m.until = until
		return true, nil
	}
	return false, ErrExtendFailed
}

func genValue() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (m *Mutex) acquire(ctx context.Context, client redis.UniversalClient, value string) (bool, error) {
	reply, err := client.SetNX(ctx, m.name, value, m.expiry).Result()
	if err != nil {
		return false, err
	}
	return reply, nil
}

var deleteScript = NewScript(1, `
	local val = redis.call("GET", KEYS[1])
	if val == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	elseif val == false then
		return -1
	else
		return 0
	end
`)

func (m *Mutex) release(ctx context.Context, client redis.UniversalClient, value string) (bool, error) {
	status, err := m.Eval(ctx, client, deleteScript, m.name, value)
	if err != nil {
		return false, err
	}
	if status == int64(-1) {
		return false, ErrLockAlreadyExpired
	}
	return status != int64(0), nil
}

func (m *Mutex) Eval(ctx context.Context, client redis.UniversalClient, script *Script, keysAndArgs ...interface{}) (interface{}, error) {
	keys := make([]string, script.KeyCount)
	args := keysAndArgs

	if script.KeyCount > 0 {
		for i := 0; i < script.KeyCount; i++ {
			keys[i] = keysAndArgs[i].(string)
		}
		args = keysAndArgs[script.KeyCount:]
	}

	v, err := client.EvalSha(ctx, script.Hash, keys, args...).Result()
	if err != nil && strings.Contains(err.Error(), "NOSCRIPT ") {
		v, err = client.Eval(ctx, script.Src, keys, args...).Result()
	}
	return v, noErrNil(err)
}

var touchWithSetNXScript = NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("PEXPIRE", KEYS[1], ARGV[2])
	elseif redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2], "NX") then
		return 1
	else
		return 0
	end
`)

var touchScript = NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("PEXPIRE", KEYS[1], ARGV[2])
	else
		return 0
	end
`)

func (m *Mutex) touch(ctx context.Context, client redis.UniversalClient, value string, expiry int) (bool, error) {
	touchScript := touchScript
	if m.setNXOnExtend {
		touchScript = touchWithSetNXScript
	}

	status, err := m.Eval(ctx, client, touchScript, m.name, value, expiry)
	if err != nil {
		return false, err
	}
	return status != int64(0), nil
}

func (m *Mutex) actOnPoolsAsync(actFn func(client redis.UniversalClient) (bool, error)) (int, error) {
	type result struct {
		node     int
		statusOK bool
		err      error
	}

	ch := make(chan result, len(m.pools))
	for node, client := range m.pools {
		go func(node int, client redis.UniversalClient) {
			r := result{node: node}
			r.statusOK, r.err = actFn(client)
			ch <- r
		}(node, client)
	}

	var (
		n     = 0
		taken []int
		err   error
	)

	for range m.pools {
		r := <-ch
		if r.statusOK {
			n++
		} else if r.err == ErrLockAlreadyExpired {
			err = multierror.Append(err, ErrLockAlreadyExpired)
		} else if r.err != nil {
			err = multierror.Append(err, fmt.Errorf("node #%d: %w", r.node, r.err))
		} else {
			taken = append(taken, r.node)
			err = multierror.Append(err, fmt.Errorf("node #%d: lock already taken", r.node))
		}

		if m.failFast {
			// fast return
			if n >= m.quorum {
				return n, err
			}

			// fail fast
			if len(taken) >= m.quorum {
				return n, fmt.Errorf("lock already taken, locked nodes: %v", taken)
			}
		}
	}

	if len(taken) >= m.quorum {
		return n, fmt.Errorf("lock already taken, locked nodes: %v", taken)
	}
	return n, err
}

const (
	minRetryDelayMilliSec = 50
	maxRetryDelayMilliSec = 250
)

// Script encapsulates the source, hash and key count for a Lua script.
type Script struct {
	KeyCount int
	Src      string
	Hash     string
}

// NewScript returns a new script object. If keyCount is greater than or equal
// to zero, then the count is automatically inserted in the EVAL command
// argument list. If keyCount is less than zero, then the application supplies
// the count as the first value in the keysAndArgs argument to the Do, Send and
// SendHash methods.
func NewScript(keyCount int, src string) *Script {
	h := sha1.New()
	_, _ = io.WriteString(h, src)
	return &Script{keyCount, src, hex.EncodeToString(h.Sum(nil))}
}

func noErrNil(err error) error {
	if err != redis.Nil {
		return err
	}
	return nil
}

// An MuxOption configures a mutex.
type MuxOption interface {
	Apply(*Mutex)
}

// OptionFunc is a function that configures a mutex.
type OptionFunc func(*Mutex)

// Apply calls f(mutex)
func (f OptionFunc) Apply(mutex *Mutex) {
	f(mutex)
}

// WithExpiry can be used to set the expiry of a mutex to the given value.
// The default is 8s.
func WithExpiry(expiry time.Duration) MuxOption {
	return OptionFunc(func(m *Mutex) {
		m.expiry = expiry
	})
}

// WithTries can be used to set the number of times lock acquire is attempted.
// The default value is 32.
func WithTries(tries int) MuxOption {
	return OptionFunc(func(m *Mutex) {
		m.tries = tries
	})
}

// WithRetryDelay can be used to set the amount of time to wait between retries.
// The default value is rand(50ms, 250ms).
func WithRetryDelay(delay time.Duration) MuxOption {
	return OptionFunc(func(m *Mutex) {
		m.delayFunc = func(tries int) time.Duration {
			return delay
		}
	})
}

// WithSetNXOnExtend improves extending logic to extend the key if exist
// and if not, tries to set a new key in redis
// Useful if your redises restart often and you want to reduce the chances of losing the lock
func WithSetNXOnExtend() MuxOption {
	return OptionFunc(func(m *Mutex) {
		m.setNXOnExtend = true
	})
}

// WithRetryDelayFunc can be used to override default delay behavior.
func WithRetryDelayFunc(delayFunc DelayFunc) MuxOption {
	return OptionFunc(func(m *Mutex) {
		m.delayFunc = delayFunc
	})
}

// WithDriftFactor can be used to set the clock drift factor.
// The default value is 0.01.
func WithDriftFactor(factor float64) MuxOption {
	return OptionFunc(func(m *Mutex) {
		m.driftFactor = factor
	})
}

// WithTimeoutFactor can be used to set the timeout factor.
// The default value is 0.05.
func WithTimeoutFactor(factor float64) MuxOption {
	return OptionFunc(func(m *Mutex) {
		m.timeoutFactor = factor
	})
}

// WithGenValueFunc can be used to set the custom value generator.
func WithGenValueFunc(genValueFunc func() (string, error)) MuxOption {
	return OptionFunc(func(m *Mutex) {
		m.genValueFunc = genValueFunc
	})
}

// WithValue can be used to assign the random value without having to call lock.
// This allows the ownership of a lock to be "transferred" and allows the lock to be unlocked from elsewhere.
func WithValue(v string) MuxOption {
	return OptionFunc(func(m *Mutex) {
		m.value = v
	})
}

// WithFailFast can be used to quickly acquire and release the lock.
// When some Redis servers are blocking, we do not need to wait for responses from all the Redis servers response.
// As long as the quorum is met, we can assume the lock is acquired. The effect of this parameter is to achieve low
// latency, avoid Redis blocking causing Lock/Unlock to not return for a long time.
func WithFailFast(b bool) MuxOption {
	return OptionFunc(func(m *Mutex) {
		m.failFast = b
	})
}

// WithShufflePools can be used to shuffle Redis pools to reduce centralized access in concurrent scenarios.
func WithShufflePools(b bool) MuxOption {
	return OptionFunc(func(m *Mutex) {
		m.shuffle = b
	})
}
