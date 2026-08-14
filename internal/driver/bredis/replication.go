package bredis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	WaitModeReplication = "wait"
	WaitModeAOF         = "waitaof"
)

var (
	ErrAmbiguousCommit = errors.New("redis write commit is ambiguous")
	ErrClusterRouting  = errors.New("redis cluster routing failed")
)

type ReplicationNotConfirmedError struct {
	Command      string
	AOFLocal     int
	Required     int
	Acknowledged int64
	Timeout      time.Duration
	Cause        error
}

func (e *ReplicationNotConfirmedError) Error() string {
	command := e.Command
	if command == "" {
		command = "WAIT"
	}
	message := fmt.Sprintf("redis %s acknowledged by %d replicas, want %d within %s", command, e.Acknowledged, e.Required, e.Timeout)
	if command == "WAITAOF" {
		message = fmt.Sprintf("redis WAITAOF required %d local AOF confirmations and acknowledged by %d replicas, want %d within %s", e.AOFLocal, e.Acknowledged, e.Required, e.Timeout)
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", message, e.Cause)
	}
	return message
}

func (e *ReplicationNotConfirmedError) Unwrap() []error {
	errList := []error{ErrAmbiguousCommit}
	if e.Cause != nil {
		errList = append(errList, e.Cause)
	}
	return errList
}

type ClusterRoutingError struct {
	Key   string
	Cause error
}

func (e *ClusterRoutingError) Error() string {
	return fmt.Sprintf("redis cluster routing failed for key %q: %v", e.Key, e.Cause)
}

func (e *ClusterRoutingError) Unwrap() []error {
	return []error{ErrClusterRouting, e.Cause}
}

type replicationWait struct {
	mode     string
	replicas int
	aofLocal int
	timeout  time.Duration
}

type keyedPipeliner interface {
	PipelinedForKey(context.Context, string, string, bool, func(redis.Pipeliner) error) ([]redis.Cmder, error)
}

type redisDoer interface {
	Do(context.Context, ...any) *redis.Cmd
}

func firstReplicationWait(waits []replicationWait) replicationWait {
	if len(waits) == 0 {
		return replicationWait{}
	}
	return waits[0]
}

func (w replicationWait) enabled() bool {
	return (w.mode == WaitModeReplication && w.replicas > 0) ||
		(w.mode == WaitModeAOF && (w.aofLocal > 0 || w.replicas > 0))
}

func (w replicationWait) command() string {
	if w.mode == WaitModeAOF {
		return "WAITAOF"
	}
	return "WAIT"
}

func (w replicationWait) execute(ctx context.Context, client redis.UniversalClient, key string, write func(redis.Pipeliner) redis.Cmder) (redis.Cmder, error) {
	commands, err := w.executeMany(ctx, client, key, func(pipe redis.Pipeliner) []redis.Cmder {
		return []redis.Cmder{write(pipe)}
	})
	if len(commands) == 0 {
		return nil, err
	}
	return commands[0], err
}

func (w replicationWait) executeMany(ctx context.Context, client redis.UniversalClient, key string, write func(redis.Pipeliner) []redis.Cmder) ([]redis.Cmder, error) {
	var writeCmds []redis.Cmder
	var waitCmd *redis.Cmd
	run := func(targetAddress string, asking bool) error {
		writeCmds = nil
		waitCmd = nil
		_, err := pipelineForKey(ctx, client, key, targetAddress, asking, func(pipe redis.Pipeliner) error {
			writeCmds = write(pipe)
			switch w.mode {
			case WaitModeReplication:
				waitCmd = pipe.Do(ctx, "WAIT", w.replicas, waitTimeoutMilliseconds(w.timeout))
			case WaitModeAOF:
				waitCmd = pipe.Do(ctx, "WAITAOF", w.aofLocal, w.replicas, waitTimeoutMilliseconds(w.timeout))
			}
			return nil
		})
		return err
	}

	err := run("", false)
	redirectCmd := firstRedirect(writeCmds)
	if kind, address, ok := clusterRedirect(redirectCmd); ok {
		switch kind {
		case "MOVED":
			if cluster := clusterClient(client); cluster != nil {
				cluster.ReloadState(ctx)
			}
			err = run(address, false)
		case "ASK":
			err = run(address, true)
		}
	}
	if redirectCmd = firstRedirect(writeCmds); redirectCmd != nil {
		return nil, &ClusterRoutingError{Key: key, Cause: redirectCmd.Err()}
	}
	if len(writeCmds) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("redis write commands were not queued")
	}
	for _, cmd := range writeCmds {
		if cmd == nil || cmd.Err() == nil {
			continue
		}
		writeErr := cmd.Err()
		if _, ok := errors.AsType[redis.Error](writeErr); ok {
			return nil, writeErr
		}
		return nil, w.notConfirmed(writeErr)
	}
	if waitCmd == nil {
		return nil, fmt.Errorf("redis %s command was not queued", w.command())
	}
	if waitErr := w.waitResult(waitCmd); waitErr != nil {
		return nil, waitErr
	}
	if err != nil {
		return nil, w.notConfirmed(err)
	}
	return writeCmds, nil
}

func (w replicationWait) confirm(ctx context.Context, client redisDoer) error {
	var command *redis.Cmd
	if w.mode == WaitModeAOF {
		command = client.Do(ctx, "WAITAOF", w.aofLocal, w.replicas, waitTimeoutMilliseconds(w.timeout))
	} else {
		command = client.Do(ctx, "WAIT", w.replicas, waitTimeoutMilliseconds(w.timeout))
	}
	return w.waitResult(command)
}

func (w replicationWait) waitResult(waitCmd *redis.Cmd) error {
	if w.mode == WaitModeAOF {
		acknowledged, err := waitCmd.Int64Slice()
		if err != nil {
			return w.notConfirmed(err)
		}
		if len(acknowledged) != 2 {
			return w.notConfirmed(fmt.Errorf("unexpected WAITAOF response length %d", len(acknowledged)))
		}
		if acknowledged[0] < int64(w.aofLocal) {
			return w.notConfirmed(fmt.Errorf("WAITAOF acknowledged by %d local AOF instances, want %d", acknowledged[0], w.aofLocal))
		}
		return w.validateAcknowledged(acknowledged[1])
	}

	acknowledged, err := waitCmd.Int64()
	if err != nil {
		return w.notConfirmed(err)
	}
	return w.validateAcknowledged(acknowledged)
}

func (w replicationWait) validateAcknowledged(acknowledged int64) error {
	if acknowledged < int64(w.replicas) {
		return &ReplicationNotConfirmedError{Command: w.command(), Required: w.replicas, AOFLocal: w.aofLocal, Acknowledged: acknowledged, Timeout: w.timeout}
	}
	return nil
}

func (w replicationWait) notConfirmed(cause error) error {
	return &ReplicationNotConfirmedError{Command: w.command(), Required: w.replicas, AOFLocal: w.aofLocal, Timeout: w.timeout, Cause: cause}
}

func firstRedirect(commands []redis.Cmder) redis.Cmder {
	for _, command := range commands {
		if _, _, ok := clusterRedirect(command); ok {
			return command
		}
	}
	return nil
}

func waitTimeoutMilliseconds(timeout time.Duration) int64 {
	if timeout > 0 && timeout < time.Millisecond {
		return 1
	}
	return timeout.Milliseconds()
}

func pipelineForKey(ctx context.Context, client redis.UniversalClient, key, targetAddress string, asking bool, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	if keyed, ok := client.(keyedPipeliner); ok {
		return keyed.PipelinedForKey(ctx, key, targetAddress, asking, fn)
	}
	return pipelinedForClient(ctx, client, key, targetAddress, asking, fn)
}

func pipelinedForClient(ctx context.Context, client redis.UniversalClient, key, targetAddress string, asking bool, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	var node *redis.Client
	switch client := client.(type) {
	case *redis.Client:
		node = client
	case *redis.ClusterClient:
		var err error
		node, err = client.MasterForKey(ctx, key)
		if err != nil {
			return nil, &ClusterRoutingError{Key: key, Cause: err}
		}
	default:
		return nil, fmt.Errorf("redis client %T does not support keyed pipelines", client)
	}

	target := node
	if targetAddress != "" {
		options := *node.Options()
		options.Addr = targetAddress
		options.MaxRetries = -1
		target = redis.NewClient(&options)
		defer target.Close()
	}

	conn := target.Conn()
	defer conn.Close()
	return conn.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		if asking {
			pipe.Do(ctx, "ASKING")
		}
		return fn(pipe)
	})
}

func clusterClient(client redis.UniversalClient) *redis.ClusterClient {
	if holder, ok := client.(*RedisHolder); ok {
		client = holder.UniversalClient
	}
	cluster, _ := client.(*redis.ClusterClient)
	return cluster
}

func clusterRedirect(cmd redis.Cmder) (kind, address string, ok bool) {
	if cmd == nil {
		return "", "", false
	}
	if address, ok := redis.IsMovedError(cmd.Err()); ok {
		return "MOVED", address, true
	}
	if address, ok := redis.IsAskError(cmd.Err()); ok {
		return "ASK", address, true
	}
	return "", "", false
}
