package bredis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	bjson "github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/spf13/cast"
)

const (
	// sequenceQueueSchemaVersion identifies the Redis data-layout contract and prevents incompatible queue implementations from sharing data.
	sequenceQueueSchemaVersion = "3"
	// sequenceQueueDefaultMaxLen is retained for source compatibility.
	sequenceQueueDefaultMaxLen = partitionQueueDefaultMaxLen
)

type sequenceQueueStore struct {
	client   redis.UniversalClient
	scripts  *ScriptCatalog
	topology sequenceQueueTopology
	maxLen   int64
	lease    time.Duration
}

type sequenceQueueToken struct {
	Partition   int64
	SchedulerID string
	OrderKey    string
}

type sequenceQueueEnqueueResult struct {
	Code        string
	SchedulerID string
	Pending     int64
}

type sequenceQueueAcquireResult struct {
	Code          string
	Head          string
	AcquisitionID string
	DeadlineMS    int64
}

type sequenceQueueHeartbeatResult struct {
	Code          string
	AcquisitionID string
	DeadlineMS    int64
}

type sequenceQueueFinalizeResult struct {
	Code        string
	SchedulerID string
	Remaining   int64
}

type sequenceQueueAutoClaimResult struct {
	Tokens []sequenceQueueToken
	Cursor string
}

func newSequenceQueueStore(client redis.UniversalClient, topology sequenceQueueTopology, maxLen int64, lease time.Duration) *sequenceQueueStore {
	if maxLen <= 0 {
		maxLen = sequenceQueueDefaultMaxLen
	}
	if lease <= 0 {
		lease = time.Minute
	}
	return &sequenceQueueStore{client: client, scripts: DefaultScriptCatalog(), topology: topology, maxLen: maxLen, lease: lease}
}

func (s *sequenceQueueStore) canonicalConfig() string {
	return s.metadata().canonicalConfig()
}

func (s *sequenceQueueStore) metadata() partitionQueueMetadata {
	return partitionQueueMetadata{schema: sequenceQueueSchemaVersion, partitions: s.topology.partitions, capacity: s.maxLen}
}

// ensureMetadata elects the first complete queue configuration as canonical.
func (s *sequenceQueueStore) ensureMetadata(ctx context.Context) error {
	return ensurePartitionQueueMetadata(ctx, s.client, s.topology.metadataKey(), "sequence queue", s.metadata())
}

func validateSequenceQueueMetadata(got, want string) error {
	return validatePartitionQueueMetadata(got, want, "sequence queue")
}

func (s *sequenceQueueStore) bootstrapGroups(ctx context.Context, group string) error {
	return bootstrapPartitionGroups(ctx, s.client, group, "sequence queue", s.topology.partitions, s.topology.schedulerKey)
}

func (s *sequenceQueueStore) enqueue(ctx context.Context, orderKey string, data map[string]any) (sequenceQueueEnqueueResult, error) {
	if orderKey == "" {
		return sequenceQueueEnqueueResult{}, errors.New("missing orderKey")
	}
	payload, err := bjson.Marshal(data)
	if err != nil {
		return sequenceQueueEnqueueResult{}, fmt.Errorf("marshal sequence queue message: %w", err)
	}
	partition := s.topology.partition(orderKey)
	values, err := s.runTriplet(ctx, ScriptSequenceQueueEnqueue, []string{
		s.topology.orderListKey(partition, orderKey), s.topology.orderStateKey(partition, orderKey),
		s.topology.schedulerKey(partition), s.topology.partitionStateKey(partition), s.topology.isolationKey(partition),
	}, string(payload), orderKey, s.maxLen)
	if err != nil {
		return sequenceQueueEnqueueResult{}, fmt.Errorf("enqueue sequence queue message: %w", err)
	}
	return sequenceQueueEnqueueResult{Code: values[0], SchedulerID: values[1], Pending: cast.ToInt64(values[2])}, nil
}

func (s *sequenceQueueStore) readOne(ctx context.Context, group, consumer string, partition int64, block time.Duration) (*sequenceQueueToken, error) {
	stream := s.topology.schedulerKey(partition)
	streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: group, Consumer: consumer, Streams: []string{stream, ">"}, Count: 1, Block: block}).Result()
	if err != nil {
		return nil, err
	}
	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, redis.Nil
	}
	message := streams[0].Messages[0]
	token, err := tokenFromMessage(partition, message)
	if err != nil {
		return nil, errors.Join(err, s.discardSchedulerToken(ctx, stream, group, message.ID))
	}
	return token, nil
}

// autoClaim supports resumable, bounded scans. Invalid tokens are discarded and
// omitted from the returned batch rather than poisoning the claim cursor.
func (s *sequenceQueueStore) autoClaim(ctx context.Context, group, consumer string, partition int64, minIdle time.Duration, cursor string, count int64) (sequenceQueueAutoClaimResult, error) {
	if cursor == "" {
		cursor = "0-0"
	}
	if count <= 0 {
		count = 1
	}
	stream := s.topology.schedulerKey(partition)
	messages, next, err := s.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{Stream: stream, Group: group, Consumer: consumer, MinIdle: minIdle, Start: cursor, Count: count}).Result()
	result := sequenceQueueAutoClaimResult{Cursor: next}
	if err != nil {
		return result, err
	}
	for _, message := range messages {
		token, tokenErr := tokenFromMessage(partition, message)
		if tokenErr != nil {
			if discardErr := s.discardSchedulerToken(ctx, stream, group, message.ID); discardErr != nil {
				return result, errors.Join(tokenErr, discardErr)
			}
			continue
		}
		result.Tokens = append(result.Tokens, *token)
	}
	return result, nil
}

func (s *sequenceQueueStore) autoClaimOne(ctx context.Context, group, consumer string, partition int64, minIdle time.Duration) (*sequenceQueueToken, string, error) {
	result, err := s.autoClaim(ctx, group, consumer, partition, minIdle, "0-0", 1)
	if err != nil {
		return nil, result.Cursor, err
	}
	if len(result.Tokens) == 0 {
		return nil, result.Cursor, redis.Nil
	}
	return &result.Tokens[0], result.Cursor, nil
}

func (s *sequenceQueueStore) discardSchedulerToken(ctx context.Context, stream, group, id string) error {
	return ackAndDelete(ctx, s.client, stream, group, id)
}

func tokenFromMessage(partition int64, message redis.XMessage) (*sequenceQueueToken, error) {
	if len(message.Values) != 1 {
		return nil, fmt.Errorf("scheduler token %s has non-canonical fields", message.ID)
	}
	orderKey, ok := message.Values["order_key"]
	if !ok || cast.ToString(orderKey) == "" {
		return nil, fmt.Errorf("scheduler token %s missing order_key", message.ID)
	}
	return &sequenceQueueToken{Partition: partition, SchedulerID: message.ID, OrderKey: cast.ToString(orderKey)}, nil
}

// lease acquires or renews ownership using the strict scheduler/order-key/acquisition tuple.
func (s *sequenceQueueStore) leaseToken(ctx context.Context, operation, group, consumer, acquisitionID string, token sequenceQueueToken) (sequenceQueueAcquireResult, error) {
	if err := s.validateToken(token); err != nil {
		return sequenceQueueAcquireResult{}, err
	}
	values, err := s.runTriplet(ctx, ScriptSequenceQueueLease, []string{
		s.topology.orderListKey(token.Partition, token.OrderKey), s.topology.orderStateKey(token.Partition, token.OrderKey),
		s.topology.schedulerKey(token.Partition), s.topology.partitionStateKey(token.Partition), s.topology.isolationKey(token.Partition),
	}, operation, group, token.SchedulerID, token.OrderKey, consumer, acquisitionID, s.lease.Milliseconds())
	if err != nil {
		return sequenceQueueAcquireResult{}, fmt.Errorf("%s sequence queue lease: %w", operation, err)
	}
	return sequenceQueueAcquireResult{Code: values[0], Head: values[1], AcquisitionID: acquisitionID, DeadlineMS: cast.ToInt64(values[2])}, nil
}

func (s *sequenceQueueStore) acquireWithID(ctx context.Context, group, consumer, acquisitionID string, token sequenceQueueToken) (sequenceQueueAcquireResult, error) {
	return s.leaseToken(ctx, "acquire", group, consumer, acquisitionID, token)
}

func (s *sequenceQueueStore) renew(ctx context.Context, group, consumer, acquisitionID string, token sequenceQueueToken) (sequenceQueueHeartbeatResult, error) {
	result, err := s.leaseToken(ctx, "renew", group, consumer, acquisitionID, token)
	return sequenceQueueHeartbeatResult{Code: result.Code, AcquisitionID: acquisitionID, DeadlineMS: result.DeadlineMS}, err
}

// Legacy runtime adapters. owner is used as the acquisition fence until the
// runtime migrates to acquireWithID/renew/finalizeWithID.
func (s *sequenceQueueStore) acquire(ctx context.Context, group, owner, consumer string, token sequenceQueueToken) (sequenceQueueAcquireResult, error) {
	return s.acquireWithID(ctx, group, consumer, owner, token)
}
func (s *sequenceQueueStore) heartbeat(ctx context.Context, group, owner, consumer string, token sequenceQueueToken) (sequenceQueueHeartbeatResult, error) {
	return s.renew(ctx, group, consumer, owner, token)
}

func (s *sequenceQueueStore) finalizeWithID(ctx context.Context, group, consumer, acquisitionID string, token sequenceQueueToken, expectedHead string) (sequenceQueueFinalizeResult, error) {
	if err := s.validateToken(token); err != nil {
		return sequenceQueueFinalizeResult{}, err
	}
	values, err := s.runTriplet(ctx, ScriptSequenceQueueFinalize, []string{
		s.topology.orderListKey(token.Partition, token.OrderKey), s.topology.orderStateKey(token.Partition, token.OrderKey),
		s.topology.schedulerKey(token.Partition), s.topology.partitionStateKey(token.Partition), s.topology.isolationKey(token.Partition),
	}, group, token.SchedulerID, token.OrderKey, consumer, acquisitionID, expectedHead)
	if err != nil {
		return sequenceQueueFinalizeResult{}, fmt.Errorf("finalize sequence queue token: %w", err)
	}
	return sequenceQueueFinalizeResult{Code: values[0], SchedulerID: values[1], Remaining: cast.ToInt64(values[2])}, nil
}
func (s *sequenceQueueStore) finalize(ctx context.Context, group, owner, consumer string, token sequenceQueueToken, expectedHead string) (sequenceQueueFinalizeResult, error) {
	return s.finalizeWithID(ctx, group, consumer, owner, token, expectedHead)
}

func (s *sequenceQueueStore) validateToken(token sequenceQueueToken) error {
	if token.Partition < 0 || token.Partition >= s.topology.partitions {
		return fmt.Errorf("sequence queue token partition %d out of range", token.Partition)
	}
	if token.OrderKey == "" || token.SchedulerID == "" {
		return errors.New("sequence queue token has incomplete identity")
	}
	if want := s.topology.partition(token.OrderKey); token.Partition != want {
		return fmt.Errorf("sequence queue token partition %d does not match order-key partition %d", token.Partition, want)
	}
	return nil
}

func (s *sequenceQueueStore) runTriplet(ctx context.Context, script string, keys []string, args ...any) ([3]string, error) {
	var result [3]string
	value, err := s.scripts.Run(ctx, s.client, script, keys, args...)
	if err != nil {
		return result, err
	}
	raw, ok := value.([]interface{})
	if !ok || len(raw) != 3 {
		return result, fmt.Errorf("script %s returned malformed result %#v; want strict triplet", script, value)
	}
	for i := range result {
		result[i] = cast.ToString(raw[i])
	}
	if result[0] == "" {
		return [3]string{}, fmt.Errorf("script %s returned empty status", script)
	}
	return result, nil
}
