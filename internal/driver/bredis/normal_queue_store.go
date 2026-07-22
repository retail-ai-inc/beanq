package bredis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type normalQueueStore struct {
	client   redis.UniversalClient
	topology normalQueueTopology
	maxLen   int64
}

func newNormalQueueStore(client redis.UniversalClient, topology normalQueueTopology, maxLen int64) *normalQueueStore {
	if maxLen <= 0 {
		maxLen = partitionQueueDefaultMaxLen
	}
	return &normalQueueStore{client: client, topology: topology, maxLen: maxLen}
}

func (s *normalQueueStore) canonicalConfig() string {
	return s.metadata().canonicalConfig()
}

func (s *normalQueueStore) metadata() partitionQueueMetadata {
	return partitionQueueMetadata{schema: normalQueueSchemaVersion, partitions: s.topology.partitions, capacity: s.maxLen}
}

func (s *normalQueueStore) ensureMetadata(ctx context.Context) error {
	return ensurePartitionQueueMetadata(ctx, s.client, s.topology.metadataKey(), "normal queue", s.metadata())
}

func validateNormalQueueMetadata(got, want string) error {
	return validatePartitionQueueMetadata(got, want, "normal queue")
}

func (s *normalQueueStore) bootstrapGroups(ctx context.Context, group string) error {
	return bootstrapPartitionGroups(ctx, s.client, group, "normal queue", s.topology.partitions, s.topology.streamKey)
}

func (s *normalQueueStore) bootstrapGroup(ctx context.Context, group string, partition int64) error {
	return bootstrapPartitionGroup(ctx, s.client, group, "normal queue", partition, s.topology.streamKey(partition))
}
