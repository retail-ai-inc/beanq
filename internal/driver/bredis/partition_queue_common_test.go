package bredis

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPartitionQueueTopologyBuildsVersionedMetadataKey(t *testing.T) {
	topology := newPartitionQueueTopology("prefix", "channel", "topic", 4, "beanq-test-v2", "test_queue", "2")
	if !strings.HasPrefix(topology.metadataKey(), "prefix:channel:topic:"+topology.metadataTag()+":") {
		t.Fatalf("metadata key does not start with configured route: %q", topology.metadataKey())
	}
	if !strings.Contains(topology.metadataKey(), ":test_queue:") {
		t.Fatalf("metadata key is not versioned: %q", topology.metadataKey())
	}
	if strings.Contains(topology.metadataKey(), ":v2:") {
		t.Fatalf("metadata key repeats the version outside its hash tag: %q", topology.metadataKey())
	}
	wantMetadataKey := "prefix:channel:topic:{beanq-test-v2:" + topology.queueID + ":metadata}:test_queue:metadata"
	if topology.metadataKey() != wantMetadataKey {
		t.Fatalf("metadata key = %q, want %q", topology.metadataKey(), wantMetadataKey)
	}
	if got := topology.partition("message-id"); got < 0 || got >= topology.partitions {
		t.Fatalf("partition = %d, out of range", got)
	}
}

func TestGracefulProcessingContextSurvivesReaderCancellation(t *testing.T) {
	readerCtx, cancelReader := context.WithCancel(context.Background())
	processingCtx, cancelProcessing := gracefulProcessingContext(readerCtx)
	cancelReader()

	select {
	case <-processingCtx.Done():
		t.Fatal("processing context was canceled before the graceful shutdown deadline")
	case <-time.After(20 * time.Millisecond):
	}
	cancelProcessing()
	select {
	case <-processingCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("processing context did not stop when shutdown completed")
	}
}

func TestPartitionQueueMetadataPreservesQueueName(t *testing.T) {
	err := validatePartitionQueueMetadata("old", "new", "custom queue")
	if err == nil || !strings.Contains(err.Error(), "custom queue metadata mismatch") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
