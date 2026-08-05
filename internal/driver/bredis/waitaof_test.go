package bredis

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestReplicationWaitAOFUsesSameConnection(t *testing.T) {
	server := newWaitTestServer(t, 1)
	client := server.client()
	defer client.Close()

	wait := replicationWait{mode: WaitModeAOF, replicas: 1, aofLocal: 1, timeout: time.Second}
	_, err := wait.execute(context.Background(), client, "queue:{slot}:stream", func(pipe redis.Pipeliner) redis.Cmder {
		return pipe.XAdd(context.Background(), &redis.XAddArgs{
			Stream: "queue:{slot}:stream",
			Values: map[string]any{"id": "1"},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	commands := server.snapshot()
	for index := range commands[:len(commands)-1] {
		if commands[index].name == "XADD" && commands[index+1].name == "WAITAOF" {
			if commands[index].connection != commands[index+1].connection {
				t.Fatal("XADD and WAITAOF used different connections")
			}
			return
		}
	}
	t.Fatalf("XADD followed by WAITAOF not found in %#v", commands)
}

func TestReplicationWaitAOFInsufficientAcknowledgementsIsAmbiguous(t *testing.T) {
	server := newWaitTestServer(t, 0)
	client := server.client()
	defer client.Close()

	wait := replicationWait{mode: WaitModeAOF, replicas: 1, aofLocal: 1, timeout: 25 * time.Millisecond}
	_, err := wait.execute(context.Background(), client, "queue:{slot}:stream", func(pipe redis.Pipeliner) redis.Cmder {
		return pipe.XAdd(context.Background(), &redis.XAddArgs{
			Stream: "queue:{slot}:stream",
			Values: map[string]any{"id": "1"},
		})
	})
	if !errors.Is(err, ErrAmbiguousCommit) {
		t.Fatalf("expected ambiguous commit, got %v", err)
	}
	confirmationErr, ok := errors.AsType[*ReplicationNotConfirmedError](err)
	if !ok || confirmationErr.Command != "WAITAOF" || confirmationErr.AOFLocal != 1 {
		t.Fatalf("unexpected WAITAOF confirmation error: %#v", confirmationErr)
	}
}

func TestValidateWaitAOFTopology(t *testing.T) {
	t.Run("accepts supported Redis with AOF enabled", func(t *testing.T) {
		server := newWaitTestServer(t, 0)
		client := server.client()
		defer client.Close()
		if err := ValidateWaitAOFTopology(context.Background(), client, 0, 1); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("rejects unsupported Redis version", func(t *testing.T) {
		server := newWaitTestServer(t, 0)
		server.setVersion("7.0.15")
		client := server.client()
		defer client.Close()
		err := ValidateWaitAOFTopology(context.Background(), client, 0, 1)
		if err == nil || !strings.Contains(err.Error(), "require >= 7.2") {
			t.Fatalf("unexpected version validation error: %v", err)
		}
	})

	t.Run("rejects disabled AOF", func(t *testing.T) {
		server := newWaitTestServer(t, 0)
		server.setAOFEnabled(false)
		client := server.client()
		defer client.Close()
		err := ValidateWaitAOFTopology(context.Background(), client, 0, 1)
		if err == nil || !strings.Contains(err.Error(), "appendonly is disabled") {
			t.Fatalf("unexpected AOF validation error: %v", err)
		}
	})
}
