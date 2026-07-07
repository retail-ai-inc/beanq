package tool

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
)

type wrappedRedisClient struct {
	redis.UniversalClient
}

func (w wrappedRedisClient) Client() redis.UniversalClient {
	return w.UniversalClient
}

func TestClientFacUnwrapsRedisClientProvider(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	defer rdb.Close()

	client := ClientFac(wrappedRedisClient{UniversalClient: rdb}, "test", "node-1")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if got := client.NodeId(context.Background()); got != "node-1" {
		t.Fatalf("expected node id %q, got %q", "node-1", got)
	}
}

func TestClientFacUnsupportedClientReturnsComparableError(t *testing.T) {
	client := ClientFac(nil, "test", "")
	_, err := client.Info(context.Background())
	if !errors.Is(err, ErrUnsupportedRedisClient) {
		t.Fatalf("expected ErrUnsupportedRedisClient, got %v", err)
	}
}
