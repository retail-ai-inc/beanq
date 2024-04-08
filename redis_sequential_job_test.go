package beanq

import (
	"fmt"
	"sync"
	"testing"
)

func TestSeq(t *testing.T) {

	var wait sync.WaitGroup
	seq := newSequential()

	wait.Add(3)
	// for example
	// 3:reduce stock
	go func() {
		msg1 := NewMessage([]byte("aa"))

		seq.In("3", *msg1)
		defer wait.Done()
	}()
	// 1:create order
	go func() {
		msg2 := NewMessage([]byte("dd"))
		seq.In("1", *msg2)
		defer wait.Done()
	}()
	// 2:pay status
	go func() {
		msg3 := NewMessage([]byte("cc"))
		seq.In("2", *msg3)
		seq.In("2", *msg3)
		defer wait.Done()
	}()
	wait.Wait()
	data := seq.Sort()

	fmt.Printf("%+v \n", data)
}
