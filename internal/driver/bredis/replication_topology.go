package bredis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

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
