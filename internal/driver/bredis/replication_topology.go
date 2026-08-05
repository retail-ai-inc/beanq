package bredis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

const minimumWaitAOFVersion = "7.2"

func ValidateWaitTopology(ctx context.Context, client redis.UniversalClient, mode string, requiredReplicas, requiredLocal int) error {
	switch mode {
	case "":
		return nil
	case WaitModeReplication:
		return ValidateReplicationTopology(ctx, client, requiredReplicas)
	case WaitModeAOF:
		return ValidateWaitAOFTopology(ctx, client, requiredReplicas, requiredLocal)
	default:
		return fmt.Errorf("unsupported redis wait mode %q", mode)
	}
}

// ValidateWaitAOFTopology verifies WAITAOF support and AOF durability on every writable node.
func ValidateWaitAOFTopology(ctx context.Context, client redis.UniversalClient, requiredReplicas, requiredLocal int) error {
	switch client := client.(type) {
	case *RedisHolder:
		return ValidateWaitAOFTopology(ctx, client.UniversalClient, requiredReplicas, requiredLocal)
	case *redis.Client:
		return validateWaitAOFMaster(ctx, client, requiredReplicas, requiredLocal)
	case *redis.ClusterClient:
		return client.ForEachMaster(ctx, func(ctx context.Context, master *redis.Client) error {
			return validateWaitAOFMaster(ctx, master, requiredReplicas, requiredLocal)
		})
	default:
		return fmt.Errorf("validate redis WAITAOF topology: unsupported client %T", client)
	}
}

func validateWaitAOFMaster(ctx context.Context, master *redis.Client, requiredReplicas, requiredLocal int) error {
	address := master.Options().Addr
	serverInfo, err := master.Info(ctx, "server").Result()
	if err != nil {
		return fmt.Errorf("inspect redis server at %s: %w", address, err)
	}
	version, err := redisInfoValue(serverInfo, "redis_version")
	if err != nil {
		return fmt.Errorf("inspect redis server at %s: %w", address, err)
	}
	supported, err := redisVersionAtLeast(version, 7, 2)
	if err != nil {
		return fmt.Errorf("inspect redis server at %s: %w", address, err)
	}
	if !supported {
		return fmt.Errorf("redis master %s does not support WAITAOF: Redis %s, require >= %s", address, version, minimumWaitAOFVersion)
	}

	persistenceInfo, err := master.Info(ctx, "persistence").Result()
	if err != nil {
		return fmt.Errorf("inspect redis persistence at %s: %w", address, err)
	}
	aofEnabled, err := redisInfoValue(persistenceInfo, "aof_enabled")
	if err != nil {
		return fmt.Errorf("inspect redis persistence at %s: %w", address, err)
	}
	if aofEnabled != "1" {
		return fmt.Errorf("redis master %s cannot use WAITAOF: appendonly is disabled", address)
	}
	if requiredLocal < 0 {
		return fmt.Errorf("redis master %s has invalid WAITAOF local requirement %d", address, requiredLocal)
	}
	return validateMasterReplication(ctx, master, requiredReplicas)
}

func redisInfoValue(info, key string) (string, error) {
	prefix := key + ":"
	for _, rawLine := range strings.Split(info, "\n") {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix)), nil
		}
	}
	return "", fmt.Errorf("redis info does not contain %s", key)
}

func redisVersionAtLeast(version string, requiredMajor, requiredMinor int) (bool, error) {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return false, fmt.Errorf("invalid redis_version %q", version)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, fmt.Errorf("invalid redis_version %q: %w", version, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, fmt.Errorf("invalid redis_version %q: %w", version, err)
	}
	return major > requiredMajor || major == requiredMajor && minor >= requiredMinor, nil
}

// ValidateReplicationTopology verifies that every writable Redis node has enough
// online replicas for the configured WAIT requirement.
func ValidateReplicationTopology(ctx context.Context, client redis.UniversalClient, requiredReplicas int) error {
	if requiredReplicas <= 0 {
		return nil
	}
	switch client := client.(type) {
	case *redis.Client:
		return validateMasterReplication(ctx, client, requiredReplicas)
	case *redis.ClusterClient:
		return client.ForEachMaster(ctx, func(ctx context.Context, master *redis.Client) error {
			return validateMasterReplication(ctx, master, requiredReplicas)
		})
	default:
		return fmt.Errorf("validate redis replication topology: unsupported client %T", client)
	}
}

func validateMasterReplication(ctx context.Context, master *redis.Client, requiredReplicas int) error {
	info, err := master.Info(ctx, "replication").Result()
	if err != nil {
		return fmt.Errorf("inspect redis replication at %s: %w", master.Options().Addr, err)
	}
	role, online, err := parseReplicationInfo(info)
	if err != nil {
		return fmt.Errorf("inspect redis replication at %s: %w", master.Options().Addr, err)
	}
	if role != "master" {
		return fmt.Errorf("redis node %s is %s, want master", master.Options().Addr, role)
	}
	if online < requiredReplicas {
		return fmt.Errorf("redis master %s has insufficient online replicas: required %d, actual %d", master.Options().Addr, requiredReplicas, online)
	}
	return nil
}

func parseReplicationInfo(info string) (string, int, error) {
	role := ""
	online := 0
	connected := -1
	replicaEntries := 0
	for _, rawLine := range strings.Split(info, "\n") {
		line := strings.TrimSpace(rawLine)
		switch {
		case strings.HasPrefix(line, "role:"):
			role = strings.TrimSpace(strings.TrimPrefix(line, "role:"))
		case strings.HasPrefix(line, "connected_slaves:"):
			value := strings.TrimSpace(strings.TrimPrefix(line, "connected_slaves:"))
			count, err := strconv.Atoi(value)
			if err != nil {
				return "", 0, fmt.Errorf("invalid connected_slaves %q: %w", value, err)
			}
			connected = count
		case strings.HasPrefix(line, "slave"), strings.HasPrefix(line, "replica"):
			replicaEntries++
			if strings.Contains(line, "state=online") {
				online++
			}
		}
	}
	if role == "" {
		return "", 0, fmt.Errorf("replication INFO does not contain role")
	}
	// Older Redis-compatible servers may expose only connected_slaves.
	if replicaEntries == 0 && connected > 0 {
		online = connected
	}
	return role, online, nil
}
