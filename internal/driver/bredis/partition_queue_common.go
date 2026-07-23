package bredis

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/internal/boptions"
)

const partitionQueueDefaultMaxLen int64 = 200000

const (
	partitionReaderCount     = 1
	partitionReaderIdleDelay = 20 * time.Millisecond
	partitionClaimInterval   = time.Second
	partitionNonBlockingRead = -1 * time.Nanosecond
	queueShutdownTimeout     = 30 * time.Second
)

func gracefulProcessingContext(ctx context.Context) (context.Context, context.CancelFunc) {
	processingCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	go func() {
		<-ctx.Done()
		timer := time.NewTimer(queueShutdownTimeout)
		defer timer.Stop()
		select {
		case <-processingCtx.Done():
		case <-timer.C:
			cancel()
		}
	}()
	return processingCtx, cancel
}

func waitPartitionReader(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type partitionQueueTopology struct {
	prefix     string
	channel    string
	topic      string
	partitions int64
	queueID    string
	namespace  string
	queueType  string
	schema     string
}

func newPartitionQueueTopology(prefix, channel, topic string, partitions int64, namespace, queueType, schema string) partitionQueueTopology {
	channel, topic = normalizeQueueRoute(channel, topic)
	return partitionQueueTopology{
		prefix: prefix, channel: channel, topic: topic,
		partitions: normalizePartitionCount(partitions),
		queueID:    lengthPrefixedSHA256(prefix, channel, topic),
		namespace:  namespace, queueType: queueType, schema: schema,
	}
}

func normalizePartitionCount(partitions int64) int64 {
	if partitions <= 0 {
		return 1
	}
	return partitions
}

func normalizeQueueRoute(channel, topic string) (string, string) {
	if channel == "" {
		channel = boptions.DefaultOptions.DefaultChannel
	}
	if topic == "" {
		topic = boptions.DefaultOptions.DefaultTopic
	}
	return channel, topic
}

func (t partitionQueueTopology) partition(routeKey string) int64 {
	if t.partitions < 0 {
		t.partitions = 1
	}
	return int64(tool.HashKey([]byte(routeKey), uint64(t.partitions)))
}

func (t partitionQueueTopology) metadataTag() string {
	return fmt.Sprintf("{%s:%s:metadata}", t.namespace, t.queueID)
}

func (t partitionQueueTopology) metadataKey() string {
	return strings.Join([]string{t.prefix, t.channel, t.topic, t.metadataTag(), t.queueType, "metadata"}, ":")
}

func (t partitionQueueTopology) partitionTag(partition int64) string {
	return fmt.Sprintf("{%s:%s:p%03d}", t.namespace, t.queueID, partition)
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

type partitionQueueMetadata struct {
	schema     string
	partitions int64
	capacity   int64
}

func (m partitionQueueMetadata) canonicalConfig() string {
	return fmt.Sprintf("schema=%s;partitions=%d;capacity=%d", m.schema, m.partitions, m.capacity)
}

func ensurePartitionQueueMetadata(ctx context.Context, client redis.UniversalClient, wait replicationWait, key, queueName string, metadata partitionQueueMetadata) error {
	want := metadata.canonicalConfig()
	got, err := client.HGet(ctx, key, "canonical_config").Result()
	if err == nil {
		return validatePartitionQueueMetadata(got, want, queueName)
	}
	if !errors.Is(err, redis.Nil) {
		return fmt.Errorf("read %s metadata: %w", queueName, err)
	}
	if wait.enabled() {
		_, err = wait.execute(ctx, client, key, func(pipe redis.Pipeliner) redis.Cmder {
			return pipe.HSetNX(ctx, key, "canonical_config", want)
		})
	} else {
		err = client.HSetNX(ctx, key, "canonical_config", want).Err()
	}
	if err != nil {
		return fmt.Errorf("initialize %s metadata: %w", queueName, err)
	}
	got, err = client.HGet(ctx, key, "canonical_config").Result()
	if err != nil {
		return fmt.Errorf("read %s metadata: %w", queueName, err)
	}
	return validatePartitionQueueMetadata(got, want, queueName)
}

func validatePartitionQueueMetadata(got, want, queueName string) error {
	if got != want {
		return fmt.Errorf("%s metadata mismatch: canonical=%q requested=%q", queueName, got, want)
	}
	return nil
}

func bootstrapPartitionGroups(ctx context.Context, client redis.UniversalClient, group, queueName string, partitions int64, streamKey func(int64) string) error {
	for partition := int64(0); partition < partitions; partition++ {
		if err := bootstrapPartitionGroup(ctx, client, group, queueName, partition, streamKey(partition)); err != nil {
			return err
		}
	}
	return nil
}

func bootstrapPartitionGroup(ctx context.Context, client redis.UniversalClient, group, queueName string, partition int64, stream string) error {
	err := client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("bootstrap %s group partition %d: %w", queueName, partition, err)
	}
	return nil
}

func ackAndDelete(ctx context.Context, client redis.UniversalClient, wait replicationWait, stream, group string, ids ...string) error {
	if wait.enabled() {
		_, err := wait.executeMany(ctx, client, stream, func(pipe redis.Pipeliner) []redis.Cmder {
			return []redis.Cmder{
				pipe.XAck(ctx, stream, group, ids...),
				pipe.XDel(ctx, stream, ids...),
			}
		})
		return err
	}
	_, err := client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.XAck(ctx, stream, group, ids...)
		pipe.XDel(ctx, stream, ids...)
		return nil
	})
	return err
}
