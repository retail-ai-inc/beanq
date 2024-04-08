package beanq

import (
	"fmt"
	"testing"
	"time"

	"github.com/rs/xid"
)

func TestSeq(t *testing.T) {

	guid := xid.NewWithTime(time.Now())
	fmt.Printf("%+v \n", guid.String())
	return

	seq := newSequential()

	msg1 := NewMessage([]byte("aa"))

	seq.In("3", *msg1)
	msg2 := NewMessage([]byte("dd"))
	seq.In("1", *msg2)

	msg3 := NewMessage([]byte("cc"))
	seq.In("2", *msg3)
	seq.In("2", *msg3)
	data := seq.Sort()

	fmt.Printf("%+v \n", data)
}
