package bredis

import (
	"strings"
)

// sequenceQueueTopology is an immutable description of one logical sequence
// queue's fixed sharding topology. It owns all key construction and keeps the
// configured prefix, channel, and topic at the front of every Redis key.
type sequenceQueueTopology struct {
	partitionQueueTopology
}

func newSequenceQueueTopology(prefix, channel, topic string, partitions int64) sequenceQueueTopology {
	return sequenceQueueTopology{
		partitionQueueTopology: newPartitionQueueTopology(prefix, channel, topic, partitions, "beanq-sq-v3", "sequence_queue", sequenceQueueSchemaVersion),
	}
}

func normalizeSequenceQueuePartitionCount(partitions int64) int64 {
	return normalizePartitionCount(partitions)
}

func normalizeSequenceQueueRoute(channel, topic string) (string, string) {
	return normalizeQueueRoute(channel, topic)
}

// sequenceQueueDigest identifies a logical queue without placing untrusted
// route components in a Redis cluster hash tag. Length prefixes make the input
// encoding unambiguous.
func sequenceQueueDigest(prefix, channel, topic string) string {
	return lengthPrefixedSHA256(prefix, channel, topic)
}

func sequenceQueueOrderKeyDigest(orderKey string) string {
	return lengthPrefixedSHA256(orderKey)
}

func (t sequenceQueueTopology) partitionBaseKey(partition int64) string {
	return strings.Join([]string{t.prefix, t.channel, t.topic, t.partitionTag(partition), "sequence_queue", sequenceQueuePartitionName(partition)}, ":")
}

func (t sequenceQueueTopology) schedulerKey(partition int64) string {
	return strings.Join([]string{t.partitionBaseKey(partition), "scheduler"}, ":")
}

func (t sequenceQueueTopology) partitionStateKey(partition int64) string {
	return strings.Join([]string{t.partitionBaseKey(partition), "state"}, ":")
}

// pendingKey is retained for source compatibility; v2 stores the counter in
// the partition state hash instead of a string key.
func (t sequenceQueueTopology) pendingKey(partition int64) string {
	return t.partitionStateKey(partition)
}

func (t sequenceQueueTopology) isolationKey(partition int64) string {
	return strings.Join([]string{t.partitionBaseKey(partition), "isolation"}, ":")
}

func (t sequenceQueueTopology) orderBaseKey(partition int64, orderKey string) string {
	return strings.Join([]string{t.partitionBaseKey(partition), "order", sequenceQueueOrderKeyDigest(orderKey)}, ":")
}

func (t sequenceQueueTopology) orderListKey(partition int64, orderKey string) string {
	return strings.Join([]string{t.orderBaseKey(partition, orderKey), "list"}, ":")
}

func (t sequenceQueueTopology) orderStateKey(partition int64, orderKey string) string {
	return strings.Join([]string{t.orderBaseKey(partition, orderKey), "state"}, ":")
}

func (t sequenceQueueTopology) messageStatusKey(partition int64, orderKey, id string) string {
	return strings.Join([]string{t.orderBaseKey(partition, orderKey), "status", id}, ":")
}
