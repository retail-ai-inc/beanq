package tool

import (
	"testing"

	"github.com/cespare/xxhash/v2"
)

func TestHashKeyUsesXXHashAndReturnsPartition(t *testing.T) {
	for _, tc := range []struct {
		name  string
		id    []byte
		flake uint64
	}{
		{name: "empty", id: nil, flake: 11},
		{name: "ascii", id: []byte("message-1"), flake: 17},
		{name: "utf8", id: []byte("订单-订单"), flake: 23},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := xxhash.Sum64(tc.id) % tc.flake
			if got := HashKey(tc.id, tc.flake); got != want {
				t.Fatalf("HashKey(%q, %d) = %d, want xxhash partition %d", tc.id, tc.flake, got, want)
			}
		})
	}
}

func TestHashKeyReturnsZeroForZeroFlake(t *testing.T) {
	if got := HashKey([]byte("message-1"), 0); got != 0 {
		t.Fatalf("HashKey with zero flake = %d, want 0", got)
	}
}

func TestHashKeyIsStableAndWithinPartitionRange(t *testing.T) {
	const flake = uint64(31)
	id := []byte("stable-message")
	first := HashKey(id, flake)
	if first >= flake {
		t.Fatalf("HashKey(%q, %d) = %d, outside partition range", id, flake, first)
	}
	for i := 0; i < 10; i++ {
		if got := HashKey(id, flake); got != first {
			t.Fatalf("HashKey(%q, %d) changed from %d to %d", id, flake, first, got)
		}
	}
}

func TestLogicLogShardKeysAreStableAndDistinct(t *testing.T) {
	seen := make(map[string]struct{}, BeanqLogicLogPartitions)
	for shard := uint64(0); shard < BeanqLogicLogPartitions; shard++ {
		key := MakeLogicShardKey("prefix", shard)
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate logic-log shard key %q", key)
		}
		seen[key] = struct{}{}
	}
	if got, want := MakeLogicKeyForID("prefix", "message-1"), MakeLogicKeyForID("prefix", "message-1"); got != want {
		t.Fatalf("logic-log key is not stable: %q != %q", got, want)
	}
	if got := MakeLogicShardKey("prefix", BeanqLogicLogPartitions); got != MakeLogicShardKey("prefix", 0) {
		t.Fatalf("out-of-range shard did not wrap to shard zero: %q", got)
	}
}
