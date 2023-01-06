package base

import (
	"strings"

	"beanq/internal/options"
)

func makeKey(group, queue, name string) string {

	if group == "" {
		group = options.DefaultOptions.DefaultGroup
	}
	if queue == "" {
		queue = options.DefaultOptions.DefaultQueueName
	}
	var sb strings.Builder

	sb.WriteString(group)
	sb.WriteString(":")
	sb.WriteString(queue)
	sb.WriteString(":")
	sb.WriteString(name)

	return sb.String()
}
func MakeListKey(group, queue string) string {
	return makeKey(group, queue, "list")
}
func MakeZSetKey(group, queue string) string {
	return makeKey(group, queue, "zset")
}
func MakeStreamKey(group, queue string) string {
	return makeKey(group, queue, "stream")
}
