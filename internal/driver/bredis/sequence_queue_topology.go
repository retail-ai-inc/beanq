package bredis

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/internal/boptions"
)

// sequenceQueueTopology is an immutable description of one logical sequence
// queue's fixed sharding topology. It owns all key construction so its internal
// Redis hash tag appears before caller-controlled text can introduce braces.
type sequenceQueueTopology struct {
	prefix     string
	channel    string
	topic      string
	partitions int64
	queueID    string
}

func newSequenceQueueTopology(prefix, channel, topic string, partitions int64) sequenceQueueTopology {
	channel, topic = normalizeSequenceQueueRoute(channel, topic)
	return sequenceQueueTopology{
		prefix:     prefix,
		channel:    channel,
		topic:      topic,
		partitions: normalizeSequenceQueuePartitionCount(partitions),
		queueID:    sequenceQueueDigest(prefix, channel, topic),
	}
}

func normalizeSequenceQueuePartitionCount(partitions int64) int64 {
	if partitions <= 0 {
		return 1
	}
	return partitions
}

func normalizeSequenceQueueRoute(channel, topic string) (string, string) {
	if channel == "" {
		channel = boptions.DefaultOptions.DefaultChannel
	}
	if topic == "" {
		topic = boptions.DefaultOptions.DefaultTopic
	}
	return channel, topic
}

func (t sequenceQueueTopology) partition(customerID string) int64 {
	return int64(tool.HashKey([]byte(customerID), uint64(t.partitions)))
}

// sequenceQueueDigest identifies a logical queue without placing untrusted
// route components in a Redis cluster hash tag. Length prefixes make the input
// encoding unambiguous.
func sequenceQueueDigest(prefix, channel, topic string) string {
	return lengthPrefixedSHA256(prefix, channel, topic)
}

func sequenceQueueCustomerDigest(customerID string) string {
	return lengthPrefixedSHA256(customerID)
}

func lengthPrefixedSHA256(parts ...string) string {
	h := sha256.New()
	var length [8]byte
	for _, part := range parts {
		binary.BigEndian.PutUint64(length[:], uint64(len(part)))
		_, _ = h.Write(length[:])
		_, _ = h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (t sequenceQueueTopology) metadataKey() string {
	return strings.Join([]string{t.metadataTag(), t.prefix, "sequence_queue", t.queueID, "metadata"}, ":")
}

func (t sequenceQueueTopology) metadataTag() string {
	return fmt.Sprintf("{beanq-sq:%s:metadata}", t.queueID)
}

func (t sequenceQueueTopology) partitionTag(partition int64) string {
	return fmt.Sprintf("{beanq-sq:%s:p%03d}", t.queueID, partition)
}

func (t sequenceQueueTopology) partitionBaseKey(partition int64) string {
	return strings.Join([]string{t.partitionTag(partition), t.prefix, t.channel, t.topic, "sequence_queue", sequenceQueuePartitionName(partition)}, ":")
}

func (t sequenceQueueTopology) schedulerKey(partition int64) string {
	return strings.Join([]string{t.partitionBaseKey(partition), "scheduler"}, ":")
}

func (t sequenceQueueTopology) pendingKey(partition int64) string {
	return strings.Join([]string{t.partitionBaseKey(partition), "pending"}, ":")
}

func (t sequenceQueueTopology) customerBaseKey(partition int64, customerID string) string {
	return strings.Join([]string{t.partitionBaseKey(partition), "customer", sequenceQueueCustomerDigest(customerID)}, ":")
}

func (t sequenceQueueTopology) customerListKey(partition int64, customerID string) string {
	return strings.Join([]string{t.customerBaseKey(partition, customerID), "list"}, ":")
}

func (t sequenceQueueTopology) customerStateKey(partition int64, customerID string) string {
	return strings.Join([]string{t.customerBaseKey(partition, customerID), "state"}, ":")
}
