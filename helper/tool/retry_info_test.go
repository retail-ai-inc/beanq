package tool

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryInfoReturnsLastErrorWhenContextExpiresDuringRetryWait(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	want := errors.New("handler failed")
	_, err := RetryInfo(ctx, func() error { return want }, 1)
	if !errors.Is(err, want) {
		t.Fatalf("expected last handler error, got %v", err)
	}
}
