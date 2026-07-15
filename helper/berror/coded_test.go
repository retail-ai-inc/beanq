package berror

import (
	"errors"
	"testing"
)

func TestCodedErrorMatchesByCode(t *testing.T) {
	cause := errors.New("cause")
	err := NewCodedError("broker.failed", "broker failed", cause)
	if !errors.Is(err, NewCodedError("broker.failed", "different message", nil)) {
		t.Fatal("expected coded errors to match by code")
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected coded error to unwrap its cause")
	}
	if err.BQError() != "broker.failed" {
		t.Fatalf("BQError = %q", err.BQError())
	}
}

func TestBqErrorRemainsComparable(t *testing.T) {
	if !errors.Is(ErrIdempotent, BqError("duplicate id")) {
		t.Fatal("expected BqError values to remain comparable")
	}
}
