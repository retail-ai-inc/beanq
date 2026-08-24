package bredis

import (
	"strings"
	"testing"
)

func TestHashAndListScriptsUseBulkCommands(t *testing.T) {
	tests := []struct {
		name   string
		script string
		call   string
		guard  string
	}{
		{
			name:   "save hash",
			script: saveHsetLua,
			call:   "redis.call('HSET', key, unpack(ARGV))",
			guard:  "if #ARGV > 0 then",
		},
		{
			name:   "save branches",
			script: saveBranchesLua,
			call:   "redis.call('RPUSH', KEYS[2], unpack(ARGV, 5, table.getn(ARGV)))",
		},
		{
			name:   "save transaction",
			script: saveNewTransLua,
			call:   "redis.call('RPUSH', KEYS[2], unpack(ARGV, 7, table.getn(ARGV)))",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !strings.Contains(test.script, test.call) {
				t.Fatalf("script does not contain bulk call %q", test.call)
			}
			if test.guard != "" && !strings.Contains(test.script, test.guard) {
				t.Fatalf("script does not contain empty-input guard %q", test.guard)
			}
		})
	}
}

func TestSequenceQueueScriptsBatchHashReads(t *testing.T) {
	tests := []struct {
		name      string
		script    string
		wantHMGet int
		wantHGet  int
	}{
		{name: "enqueue", script: sequenceQueueEnqueueLua, wantHMGet: 2},
		{name: "lease", script: sequenceQueueLeaseLua, wantHMGet: 1, wantHGet: 1},
		{name: "finalize", script: sequenceQueueFinalizeLua, wantHMGet: 1, wantHGet: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := strings.Count(test.script, "redis.call('HMGET'"); got != test.wantHMGet {
				t.Fatalf("HMGET calls = %d, want %d", got, test.wantHMGet)
			}
			if got := strings.Count(test.script, "redis.call('HGET'"); got != test.wantHGet {
				t.Fatalf("HGET calls = %d, want %d", got, test.wantHGet)
			}
		})
	}
}

func TestSequenceQueueCleanupUsesMultiKeyDelete(t *testing.T) {
	for name, script := range map[string]string{
		"lease":    sequenceQueueLeaseLua,
		"finalize": sequenceQueueFinalizeLua,
	} {
		if strings.Contains(script, "redis.call('DEL', orderState)\n") ||
			strings.Contains(script, "redis.call('DEL', orderQueue)\n") {
			t.Fatalf("%s script still deletes queue keys separately", name)
		}
		if !strings.Contains(script, "redis.call('DEL', orderState, orderQueue)") {
			t.Fatalf("%s script does not use multi-key DEL", name)
		}
	}
}
