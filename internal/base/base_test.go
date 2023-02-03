package base

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	err := Retry(func() error {
		fmt.Println("retry function body")
		return errors.New("error")
		// return nil
	}, 500*time.Millisecond)
	fmt.Println(err)
}
