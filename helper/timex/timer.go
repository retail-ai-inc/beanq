package timex

import (
	"sync"
	"time"
)

var TimerPool timerPool

type timerPool struct {
	p sync.Pool
}

func (t *timerPool) Get(duration time.Duration) *time.Timer {
	if tm, ok := t.p.Get().(*time.Timer); ok && tm != nil {
		tm.Reset(duration)
		return tm
	}
	return time.NewTimer(duration)
}

func (t *timerPool) Put(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:

		}
	}
	t.p.Put(timer)
}
