package bredis

import (
	"fmt"
	"strings"
)

const delayQueueSchemaVersion = "2"

type delayQueueTopology struct {
	partitionQueueTopology
}

func newDelayQueueTopology(prefix, channel, topic string, partitions int64) delayQueueTopology {
	return delayQueueTopology{partitionQueueTopology: newPartitionQueueTopology(
		prefix, channel, topic, partitions, "beanq-delay-v2", "delay_queue", delayQueueSchemaVersion,
	)}
}

func (t delayQueueTopology) scheduledKey(partition int64) string {
	return strings.Join([]string{t.prefix, t.channel, t.topic, t.partitionTag(partition), "delay_queue", fmt.Sprintf("scheduled%03d", partition)}, ":")
}

func (t delayQueueTopology) streamKey(partition int64) string {
	return strings.Join([]string{t.prefix, t.channel, t.topic, t.partitionTag(partition), "delay_queue", fmt.Sprintf("stream%03d", partition)}, ":")
}

func (t delayQueueTopology) deadLetterLockKey(partition int64) string {
	return strings.Join([]string{t.streamKey(partition), "dead_letter_lock"}, ":")
}
