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
	client        redis.UniversalClient
	scripts       *ScriptCatalog
	lease         time.Duration
	topology      sequenceQueueTopology
	maxLen        int64
	wait          replicationWait
	metadataCache *metadataValidationCache
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

func newSequenceQueueStore(client redis.UniversalClient, topology sequenceQueueTopology, maxLen int64, lease time.Duration, waits ...replicationWait) *sequenceQueueStore {
	if maxLen <= 0 {
		maxLen = sequenceQueueDefaultMaxLen
	}
	if lease <= 0 {
		lease = time.Minute
	}
	return &sequenceQueueStore{client: client, scripts: DefaultScriptCatalog(), topology: topology, maxLen: maxLen, lease: lease, wait: firstReplicationWait(waits)}
}

func (s *sequenceQueueStore) canonicalConfig() string {
	return s.metadata().canonicalConfig()
}

func (s *sequenceQueueStore) metadata() partitionQueueMetadata {
	return partitionQueueMetadata{schema: sequenceQueueSchemaVersion, partitions: s.topology.partitions, capacity: s.maxLen}
}

// ensureMetadata elects the first complete queue configuration as canonical.
func (s *sequenceQueueStore) ensureMetadata(ctx context.Context) error {
	return s.metadataCache.ensure(ctx, s.topology.metadataKey(), s.metadata().canonicalConfig(), func(ctx context.Context) error {
		return ensurePartitionQueueMetadata(ctx, s.client, s.wait, s.topology.metadataKey(), "sequence queue", s.metadata())
	})
}

func (s *sequenceQueueStore) bootstrapGroups(ctx context.Context, group string) error {
	return bootstrapPartitionGroups(ctx, s.client, group, "sequence queue", s.topology.partitions, s.topology.schedulerKey)
}

// enqueue for test
//
//nolint:unused
func (s *sequenceQueueStore) enqueue(ctx context.Context, orderKey string, data map[string]any) (sequenceQueueEnqueueResult, error) {
	return s.enqueueWithWait(ctx, orderKey, data, replicationWait{})
}

func (s *sequenceQueueStore) enqueueWithWait(ctx context.Context, orderKey string, data map[string]any, wait replicationWait) (sequenceQueueEnqueueResult, error) {
	if orderKey == "" {
		return sequenceQueueEnqueueResult{}, errors.New("missing orderKey")
	}
	payload, err := bjson.Marshal(data)
	if err != nil {
		return sequenceQueueEnqueueResult{}, fmt.Errorf("marshal sequence queue message: %w", err)
	}
	partition := s.topology.partition(orderKey)
	keys := []string{
		s.topology.orderListKey(partition, orderKey), s.topology.orderStateKey(partition, orderKey),
		s.topology.schedulerKey(partition), s.topology.partitionStateKey(partition), s.topology.isolationKey(partition),
	}
	if !wait.enabled() {
		values, err := s.runTriplet(ctx, ScriptSequenceQueueEnqueue, keys, string(payload), orderKey, s.maxLen)
		if err != nil {
			return sequenceQueueEnqueueResult{}, fmt.Errorf("enqueue sequence queue message: %w", err)
		}
		return sequenceQueueEnqueueResult{Code: values[0], SchedulerID: values[1], Pending: cast.ToInt64(values[2])}, nil
	}
	routingKey := s.topology.schedulerKey(partition)
	var resultCmd *redis.Cmd
	_, err = wait.execute(ctx, s.client, routingKey, func(pipe redis.Pipeliner) redis.Cmder {
		resultCmd = pipe.Eval(ctx, sequenceQueueEnqueueLua, keys, string(payload), orderKey, s.maxLen)
		return resultCmd
	})
	if err != nil {
		return sequenceQueueEnqueueResult{}, fmt.Errorf("enqueue sequence queue message: %w", err)
	}
	raw, err := resultCmd.Result()
	if err != nil {
		return sequenceQueueEnqueueResult{}, fmt.Errorf("enqueue sequence queue message: %w", err)
	}
	values, err := strictTriplet(ScriptSequenceQueueEnqueue, raw)
	if err != nil {
		return sequenceQueueEnqueueResult{}, err
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

func (s *sequenceQueueStore) discardSchedulerToken(ctx context.Context, stream, group, id string) error {
	return ackAndDelete(ctx, s.client, s.wait, stream, group, id)
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
	if s.wait.enabled() {
		source, err := sequenceQueueScriptSource(script)
		if err != nil {
			return [3]string{}, err
		}
		var resultCmd *redis.Cmd
		_, err = s.wait.execute(ctx, s.client, keys[0], func(pipe redis.Pipeliner) redis.Cmder {
			resultCmd = pipe.Eval(ctx, source, keys, args...)
			return resultCmd
		})
		if err != nil {
			return [3]string{}, err
		}
		value, err := resultCmd.Result()
		if err != nil {
			return [3]string{}, err
		}
		return strictTriplet(script, value)
	}
	value, err := s.scripts.Run(ctx, s.client, script, keys, args...)
	if err != nil {
		return [3]string{}, err
	}
	return strictTriplet(script, value)
}

func sequenceQueueScriptSource(name string) (string, error) {
	switch name {
	case ScriptSequenceQueueEnqueue:
		return sequenceQueueEnqueueLua, nil
	case ScriptSequenceQueueLease:
		return sequenceQueueLeaseLua, nil
	case ScriptSequenceQueueFinalize:
		return sequenceQueueFinalizeLua, nil
	default:
		return "", fmt.Errorf("redis script %q is not registered for replication WAIT", name)
	}
}

func strictTriplet(script string, value any) ([3]string, error) {
	var result [3]string
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
