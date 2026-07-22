package bredis

import (
	"fmt"
	"strings"
)

const normalQueueSchemaVersion = "2"

type normalQueueTopology struct {
	partitionQueueTopology
}

func newNormalQueueTopology(prefix, channel, topic string, partitions int64) normalQueueTopology {
	return normalQueueTopology{
		partitionQueueTopology: newPartitionQueueTopology(prefix, channel, topic, partitions, "beanq-normal-v2", "normal_queue", normalQueueSchemaVersion),
	}
}

func (t normalQueueTopology) streamKey(partition int64) string {
	return strings.Join([]string{t.prefix, t.channel, t.topic, t.partitionTag(partition), "normal_queue", fmt.Sprintf("stream%03d", partition)}, ":")
}

func (t normalQueueTopology) deadLetterLockKey(partition int64) string {
	return strings.Join([]string{t.streamKey(partition), "dead_letter_lock"}, ":")
}
