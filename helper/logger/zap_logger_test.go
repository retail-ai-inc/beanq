package logger

import (
	"errors"
	"testing"
	"time"
)

func TestNewLog(t *testing.T) {

	cfg := ZapLoggerConfig{
		Filename:    "",
		Level:       0,
		EncoderType: "",
		MaxSize:     0,
		MaxAge:      0,
		MaxBackups:  0,
		LocalTime:   false,
		Compress:    false,
		Pre:         "",
	}

	NewWithConfig(cfg).With("a", errors.New("aa")).Error(errors.New("berr"))

	for i := 0; i < 100; i++ {
		if i <= 10 {
			go func() {
				NewWithConfig(cfg).With("aa", 10).Error("aa info")
			}()
		}
		if i > 10 && i < 30 {
			go func() {
				NewWithConfig(cfg).With("bb", 10).With("berr", errors.New("this is an error")).Info("bb info")
			}()
		}
		if i > 30 && i < 50 {
			go func() {
				NewWithConfig(cfg).With("cc", 10).With("berr", errors.New("this is an error")).Info("cc info")
			}()
		}
		if i > 50 {
			go func() {
				NewWithConfig(cfg).With("dd", 10).With("berr", errors.New("this is an error")).Info("dd info")
			}()
		}

	}
	time.Sleep(time.Second)
}
