package beanq

import (
	"sort"
	"sync"
)

type (
	Sequential struct {
		lock    *sync.RWMutex
		orderBy string
		data    map[string]Message
	}

	ISequential interface {
		In(key string, message Message) *Sequential
		Sort() []Message
	}
)

var _ ISequential = (*Sequential)(nil)

func newSequential() *Sequential {
	return &Sequential{
		lock:    new(sync.RWMutex),
		data:    make(map[string]Message, 5),
		orderBy: "asc",
	}
}

func (t *Sequential) In(orderKey string, message Message) *Sequential {
	t.lock.RLock()
	t.data[orderKey] = message
	t.lock.RUnlock()
	return t
}

func (t *Sequential) Sort() []Message {

	d := t.data
	length := len(d)

	keys := make([]string, 0, length)
	for i, _ := range d {
		keys = append(keys, i)
	}
	sort.StringSlice(keys).Sort()

	data := make([]Message, 0, length)
	for _, v := range keys {
		data = append(data, d[v])
	}
	return data

}

func (t *Sequential) Publish() {
	// business
}
