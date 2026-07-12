package bredis

import (
	"testing"

	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
)

func TestDeadLetterXAddArgsRepublishesUntilRetryLimit(t *testing.T) {
	val := map[string]any{
		deadLetterRetryField: "3",
		"maxLen":             "10",
		"payload":            "body",
	}

	args := deadLetterXAddArgs("stream-key", "logic-key", val)
	if args.Stream != "stream-key" {
		t.Fatalf("expected republish to stream-key, got %q", args.Stream)
	}
	if got := val[deadLetterRetryField]; got != 4 {
		t.Fatalf("expected incremented deadletter retry 4, got %#v", got)
	}
	if args.MaxLen != 10 {
		t.Fatalf("expected maxLen 10, got %d", args.MaxLen)
	}
	if _, ok := val["logType"]; ok {
		t.Fatalf("did not expect logType when republishing: %#v", val)
	}
}

func TestDeadLetterXAddArgsMovesToLogicAfterRetryLimit(t *testing.T) {
	val := map[string]any{
		deadLetterRetryField: 4,
		"payload":            "body",
	}

	args := deadLetterXAddArgs("stream-key", "logic-key", val)
	if args.Stream != "logic-key" {
		t.Fatalf("expected move to logic-key, got %q", args.Stream)
	}
	if got := val[deadLetterRetryField]; got != 4 {
		t.Fatalf("expected deadletter retry to remain 4, got %#v", got)
	}
	if got := val["logType"]; got != bstatus.Dlq {
		t.Fatalf("expected DLQ logType, got %#v", got)
	}
}

func TestIncrementDeadLetterRetryDefaultsMissingValue(t *testing.T) {
	val := map[string]any{}
	if got := incrementDeadLetterRetry(val); got != 1 {
		t.Fatalf("expected retry 1, got %d", got)
	}
	if got := val[deadLetterRetryField]; got != 1 {
		t.Fatalf("expected retry stored in payload, got %#v", got)
	}
}
