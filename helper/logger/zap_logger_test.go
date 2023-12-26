package logger

import (
	"errors"
	"testing"
	"time"
)

func TestNewLog(t *testing.T) {

	for i := 0; i < 100; i++ {
		if i <= 10 {
			go func() {
				NewLogger("", 0, 0, 0, true, true, true).With("aa", 10).Info("aa info")
			}()
		}
		if i > 10 && i < 30 {
			go func() {
				NewLogger("", 0, 0, 0, true, true, true).With("bb", 10).With("err", errors.New("this is an error")).Info("bb info")
			}()
		}
		if i > 30 && i < 50 {
			go func() {
				NewLogger("", 0, 0, 0, true, true, true).With("cc", 10).With("err", errors.New("this is an error")).Info("cc info")
			}()
		}
		if i > 50 {
			go func() {
				NewLogger("", 0, 0, 0, true, true, true).With("dd", 10).With("err", errors.New("this is an error")).Info("dd info")
			}()
		}

	}
	time.Sleep(time.Second)
}
