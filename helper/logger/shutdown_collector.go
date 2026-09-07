package logger

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"sync"
)

type shutdownCollectorKey struct{}

type shutdownErrorItem struct {
	key   string
	err   error
	count int
}

type ShutdownErrorCollector struct {
	mu     sync.Mutex
	items  map[string]*shutdownErrorItem
	closed bool
}

func NewShutdownErrorCollector() *ShutdownErrorCollector {
	return &ShutdownErrorCollector{items: make(map[string]*shutdownErrorItem)}
}
func WithShutdownCollector(ctx context.Context, c *ShutdownErrorCollector) context.Context {
	return context.WithValue(ctx, shutdownCollectorKey{}, c)
}
func ShutdownCollector(ctx context.Context) *ShutdownErrorCollector {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(shutdownCollectorKey{}).(*ShutdownErrorCollector)
	return c
}

func LogRuntimeError(ctx context.Context, err error) {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}
	if c := ShutdownCollector(ctx); c != nil && ctx.Err() != nil {
		c.Add(err)
		return
	}
	New().Error(err)
}

func (c *ShutdownErrorCollector) Add(err error) {
	if c == nil || err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}
	key := normalizeShutdownError(err)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	if item, ok := c.items[key]; ok {
		item.count++
		return
	}
	c.items[key] = &shutdownErrorItem{key: key, err: err, count: 1}
}

func (c *ShutdownErrorCollector) Flush() {
	if c == nil {
		return
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	items := make([]shutdownErrorItem, 0, len(c.items))
	for _, item := range c.items {
		items = append(items, *item)
	}
	c.mu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].key < items[j].key })
	for _, item := range items {
		if item.count > 1 {
			New().Error(item.err, " (repeated ", item.count, " times)")
		} else {
			New().Error(item.err)
		}
	}
}

var shutdownDynamicPattern = regexp.MustCompile(`( partition | consumer | message )[^ :]+`)

func normalizeShutdownError(err error) string {
	return shutdownDynamicPattern.ReplaceAllString(err.Error(), `${1}*`)
}
