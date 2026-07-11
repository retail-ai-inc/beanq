package bredis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	bjson "github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/spf13/cast"
)

const (
	sequenceQueueSchemaVersion = "1"
	sequenceQueueDefaultMaxLen = 200000

	sequenceQueueCodeCreated          = "CREATED"
	sequenceQueueCodeOK               = "OK"
	sequenceQueueCodeMismatch         = "MISMATCH"
	sequenceQueueCodeScheduled        = "SCHEDULED"
	sequenceQueueCodeQueued           = "QUEUED"
	sequenceQueueCodeFull             = "FULL"
	sequenceQueueCodeAcquired         = "ACQUIRED"
	sequenceQueueCodeHeartbeat        = "HEARTBEAT"
	sequenceQueueCodeSuccessor        = "SUCCESSOR"
	sequenceQueueCodeEmpty            = "EMPTY"
	sequenceQueueCodeStaleToken       = "STALE_TOKEN"
	sequenceQueueCodeStaleOwner       = "STALE_OWNER"
	sequenceQueueCodePELOwnerMismatch = "PEL_OWNER_MISMATCH"
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
	CustomerID  string
}

type sequenceQueueEnqueueResult struct {
	Code        string
	SchedulerID string
	Pending     int64
}

type sequenceQueueAcquireResult struct {
	Code       string
	Head       string
	DeadlineMS int64
}

type sequenceQueueHeartbeatResult struct {
	Code       string
	DeadlineMS int64
}

type sequenceQueueFinalizeResult struct {
	Code        string
	SchedulerID string
	Remaining   int64
}

func newSequenceQueueStore(client redis.UniversalClient, topology sequenceQueueTopology, maxLen int64, lease time.Duration) *sequenceQueueStore {
	if maxLen <= 0 {
		maxLen = sequenceQueueDefaultMaxLen
	}
	if lease <= 0 {
		lease = time.Minute
	}
	return &sequenceQueueStore{
		client:   client,
		scripts:  DefaultScriptCatalog(),
		topology: topology,
		maxLen:   maxLen,
		lease:    lease,
	}
}

func (s *sequenceQueueStore) ensureMetadata(ctx context.Context) error {
	values, err := s.runResult(ctx, ScriptSequenceQueueMeta,
		[]string{s.topology.metadataKey()},
		sequenceQueueSchemaVersion, s.topology.partitions)
	if err != nil {
		return fmt.Errorf("ensure sequence queue metadata: %w", err)
	}
	if values[0] == sequenceQueueCodeOK || values[0] == sequenceQueueCodeCreated {
		return nil
	}
	return fmt.Errorf("sequence queue metadata %s: schema=%q partitions=%q", values[0], resultAt(values, 1), resultAt(values, 2))
}

func (s *sequenceQueueStore) bootstrapGroups(ctx context.Context, group string) error {
	for partition := int64(0); partition < s.topology.partitions; partition++ {
		stream := s.topology.schedulerKey(partition)
		err := s.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
		if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("bootstrap sequence queue group partition %d: %w", partition, err)
		}
	}
	return nil
}

func (s *sequenceQueueStore) enqueue(ctx context.Context, customerID string, data map[string]any) (sequenceQueueEnqueueResult, error) {
	if customerID == "" {
		return sequenceQueueEnqueueResult{}, errors.New("missing customerId")
	}
	payload, err := bjson.Marshal(data)
	if err != nil {
		return sequenceQueueEnqueueResult{}, fmt.Errorf("marshal sequence queue message: %w", err)
	}
	partition := s.topology.partition(customerID)
	values, err := s.runResult(ctx, ScriptSequenceQueueEnqueue, []string{
		s.topology.customerListKey(partition, customerID),
		s.topology.customerStateKey(partition, customerID),
		s.topology.schedulerKey(partition),
		s.topology.pendingKey(partition),
	}, string(payload), customerID, s.maxLen)
	if err != nil {
		return sequenceQueueEnqueueResult{}, fmt.Errorf("enqueue sequence queue message: %w", err)
	}
	return sequenceQueueEnqueueResult{
		Code:        values[0],
		SchedulerID: resultAt(values, 1),
		Pending:     cast.ToInt64(resultAt(values, 2)),
	}, nil
}

func (s *sequenceQueueStore) readOne(ctx context.Context, group, consumer string, partition int64, block time.Duration) (*sequenceQueueToken, error) {
	stream := s.topology.schedulerKey(partition)
	streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    block,
	}).Result()
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

func (s *sequenceQueueStore) autoClaimOne(ctx context.Context, group, consumer string, partition int64, minIdle time.Duration) (*sequenceQueueToken, string, error) {
	stream := s.topology.schedulerKey(partition)
	messages, next, err := s.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   stream,
		Group:    group,
		Consumer: consumer,
		MinIdle:  minIdle,
		Start:    "0-0",
		Count:    1,
	}).Result()
	if err != nil {
		return nil, next, err
	}
	if len(messages) == 0 {
		return nil, next, redis.Nil
	}
	message := messages[0]
	token, err := tokenFromMessage(partition, message)
	if err != nil {
		return nil, next, errors.Join(err, s.discardSchedulerToken(ctx, stream, group, message.ID))
	}
	return token, next, nil
}

func (s *sequenceQueueStore) discardSchedulerToken(ctx context.Context, stream, group, id string) error {
	_, err := s.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.XAck(ctx, stream, group, id)
		pipe.XDel(ctx, stream, id)
		return nil
	})
	return err
}

func tokenFromMessage(partition int64, message redis.XMessage) (*sequenceQueueToken, error) {
	customerID := cast.ToString(message.Values["customer_id"])
	if customerID == "" {
		return nil, fmt.Errorf("scheduler token %s missing customer_id", message.ID)
	}
	return &sequenceQueueToken{Partition: partition, SchedulerID: message.ID, CustomerID: customerID}, nil
}

func (s *sequenceQueueStore) acquire(ctx context.Context, group, owner, consumer string, token sequenceQueueToken) (sequenceQueueAcquireResult, error) {
	values, err := s.runResult(ctx, ScriptSequenceQueueAcquire, []string{
		s.topology.customerListKey(token.Partition, token.CustomerID),
		s.topology.customerStateKey(token.Partition, token.CustomerID),
		s.topology.schedulerKey(token.Partition),
	}, group, token.SchedulerID, token.CustomerID, owner, consumer, s.lease.Milliseconds())
	if err != nil {
		return sequenceQueueAcquireResult{}, fmt.Errorf("acquire sequence queue token: %w", err)
	}
	return sequenceQueueAcquireResult{Code: values[0], Head: resultAt(values, 1), DeadlineMS: cast.ToInt64(resultAt(values, 2))}, nil
}

func (s *sequenceQueueStore) heartbeat(ctx context.Context, group, owner, consumer string, token sequenceQueueToken) (sequenceQueueHeartbeatResult, error) {
	values, err := s.runResult(ctx, ScriptSequenceQueueHeartbeat, []string{
		s.topology.customerStateKey(token.Partition, token.CustomerID),
		s.topology.schedulerKey(token.Partition),
	}, group, token.SchedulerID, owner, consumer, s.lease.Milliseconds())
	if err != nil {
		return sequenceQueueHeartbeatResult{}, fmt.Errorf("heartbeat sequence queue token: %w", err)
	}
	return sequenceQueueHeartbeatResult{Code: values[0], DeadlineMS: cast.ToInt64(resultAt(values, 1))}, nil
}

func (s *sequenceQueueStore) finalize(ctx context.Context, group, owner, consumer string, token sequenceQueueToken, expectedHead string) (sequenceQueueFinalizeResult, error) {
	values, err := s.runResult(ctx, ScriptSequenceQueueFinalize, []string{
		s.topology.customerListKey(token.Partition, token.CustomerID),
		s.topology.customerStateKey(token.Partition, token.CustomerID),
		s.topology.schedulerKey(token.Partition),
		s.topology.pendingKey(token.Partition),
	}, group, token.SchedulerID, token.CustomerID, owner, consumer, expectedHead)
	if err != nil {
		return sequenceQueueFinalizeResult{}, fmt.Errorf("finalize sequence queue token: %w", err)
	}
	return sequenceQueueFinalizeResult{Code: values[0], SchedulerID: resultAt(values, 1), Remaining: cast.ToInt64(resultAt(values, 2))}, nil
}

func (s *sequenceQueueStore) runResult(ctx context.Context, script string, keys []string, args ...any) ([]string, error) {
	value, err := s.scripts.Run(ctx, s.client, script, keys, args...)
	if err != nil {
		return nil, err
	}
	raw, ok := value.([]interface{})
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("script %s returned malformed result %#v", script, value)
	}
	result := make([]string, len(raw))
	for i := range raw {
		result[i] = cast.ToString(raw[i])
	}
	return result, nil
}

func resultAt(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return values[index]
}
