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

func retry(f func() error, delayTime time.Duration) error {
	var index time.Duration = 0
	err := make(chan error)
	stopFlag := make(chan struct{})
	timer := time.NewTimer(index * delayTime)
	go func() {

	Loop:
		for {
			select {
			case <-timer.C:
				err <- f()

				if index == 2 {
					timer.Stop()
					stopFlag <- struct{}{}
					break Loop
				}
				if e := <-err; e == nil {
					timer.Stop()
					stopFlag <- struct{}{}
					break Loop
				}
				index++
				timer.Reset(index * delayTime)
			}
		}

	}()
	select {
	case <-stopFlag:
		for v := range err {
			close(err)
			return v
		}
	}
	close(stopFlag)
	return nil
}
func TestRe(t *testing.T) {
	err := retry(func() error {
		fmt.Println("retry function body")
		return errors.New("error")
		// return nil
	}, 500*time.Millisecond)
	fmt.Println(err)
}
