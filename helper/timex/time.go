package timex

import (
	"time"
)

// HalfHour Half-hour aligned exact times
// example:
// * 8:17 -> 8:29:59
// * 8:42 -> 8:59:59
// * 9:23 -> 9:29:59
// * 11:49 -> 11:59:59
func HalfHour(now time.Time) time.Time {

	newNow := now
	_, m, s := newNow.Clock()

	if m <= 29 {
		newNow = newNow.Add(time.Duration(29-m) * time.Minute)
	}
	if m > 29 && m <= 59 {
		newNow = newNow.Add(time.Duration(59-m) * time.Minute)
	}
	newNow = newNow.Add(time.Duration(59-s) * time.Second)

	return newNow
}
