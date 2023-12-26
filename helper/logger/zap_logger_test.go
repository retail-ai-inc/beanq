package logger

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/spf13/cast"
)

func TestNewLog(t *testing.T) {
	var a int = 10
	fmt.Println(cast.ToInt(&a))
	return

	for i := 0; i < 100; i++ {
		if i <= 10 {
			go func() {
				NewLogger().With("aa", 10).Info("aa info")
			}()
		}
		if i > 10 && i < 30 {
			go func() {
				NewLogger().With("bb", 10).With("err", errors.New("this is an error")).Info("bb info")
			}()
		}
		if i > 30 && i < 50 {
			go func() {
				NewLogger().With("cc", 10).With("err", errors.New("this is an error")).Info("cc info")
			}()
		}
		if i > 50 {
			go func() {
				NewLogger().With("dd", 10).With("err", errors.New("this is an error")).Info("dd info")
			}()
		}

	}
	time.Sleep(time.Second)
}
