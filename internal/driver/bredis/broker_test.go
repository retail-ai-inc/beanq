package bredis

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/internal/btype"
)

func TestNewBrokerWithOptionsWiresQueueRuntimeOptions(t *testing.T) {
	options := queueOptions{
		client: nil, prefix: "prefix", maxLen: 100, partitions: 7,
		runtime:        queueRuntimeOptions{workers: 3, readers: 5},
		deadLetterIdle: time.Minute, wait: replicationWait{replicas: 2, timeout: time.Second},
	}
	queues := []struct {
		name string
		base queueBase
	}{
		{name: "normal", base: newNormalWithOptions(options).base},
		{name: "delay", base: newScheduleWithOptions(options).base},
		{name: "sequence", base: newSequenceQueueWithOptions(options).base},
	}

	for _, queue := range queues {
		base := queue.base
		if base.consumerPoolSize != 3 || base.consumerReaderPoolSize != 5 {
			t.Fatalf("%s pools = (%d, %d), want (3, 5)", queue.name, base.consumerPoolSize, base.consumerReaderPoolSize)
		}
		if base.wait.replicas != 2 || base.wait.timeout != time.Second {
			t.Fatalf("%s replication wait = %#v", queue.name, base.wait)
		}
	}
}

func TestRdbBrokerRegistersEveryQueueMood(t *testing.T) {
	broker := NewBrokerWithPartitions(nil, "prefix", 100, 5, 7, 11, 2, 0)
	for _, mood := range []btype.MoodType{btype.NORMAL, btype.DELAY, btype.SEQUENCE_QUEUE} {
		route, ok := broker.routes[mood]
		if !ok || route.publish == nil || route.consume == nil {
			t.Fatalf("route %q is incomplete: %#v", mood, route)
		}
	}
}

func TestRdbBrokerRejectsUnknownQueueMood(t *testing.T) {
	broker := NewBrokerWithPartitions(nil, "prefix", 100, 5, 7, 11, 2, 0)
	if _, err := broker.Consumer(btype.MoodType("unknown"), nil); !errors.Is(err, berror.BrokerDriverError) {
		t.Fatalf("Consumer error = %v, want broker driver error", err)
	}
	if err := broker.Enqueue(context.Background(), map[string]any{"moodType": "unknown"}); !errors.Is(err, berror.BrokerDriverError) {
		t.Fatalf("Enqueue error = %v, want broker driver error", err)
	}
}

func TestIsConsumerGroupExistsError(t *testing.T) {
	if !isConsumerGroupExistsError(errors.New("BUSYGROUP Consumer Group name already exists")) {
		t.Fatal("expected BUSYGROUP to be recognized")
	}
	if isConsumerGroupExistsError(errors.New("connection refused")) {
		t.Fatal("did not expect unrelated error to be recognized")
	}
	if isConsumerGroupExistsError(nil) {
		t.Fatal("did not expect nil error to be recognized")
	}
}

func TestDeadLetterXAddArgsRepublishesUntilRetryLimit(t *testing.T) {
	val := map[string]any{
		deadLetterRetryField: "2",
		"maxLen":             "10",
		"payload":            "body",
	}

	args := (deadLetterMessage{values: val}).xAddArgs("stream-key", "logic-key")
	if args.Stream != "stream-key" {
		t.Fatalf("expected republish to stream-key, got %q", args.Stream)
	}
	if got := val[deadLetterRetryField]; got != 3 {
		t.Fatalf("expected incremented deadletter retry 3, got %#v", got)
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
		deadLetterRetryField: 3,
		"payload":            "body",
	}

	args := (deadLetterMessage{values: val}).xAddArgs("stream-key", "logic-key")
	if args.Stream != "logic-key" {
		t.Fatalf("expected move to logic-key, got %q", args.Stream)
	}
	if got := val[deadLetterRetryField]; got != 3 {
		t.Fatalf("expected deadletter retry to remain 3, got %#v", got)
	}
	if got := val["logType"]; got != bstatus.Dlq {
		t.Fatalf("expected DLQ logType, got %#v", got)
	}
	if got := val["status"]; got != bstatus.StatusFailed {
		t.Fatalf("expected failed status, got %#v", got)
	}
}

func TestIncrementDeadLetterRetryDefaultsMissingValue(t *testing.T) {
	val := map[string]any{}
	if got := (deadLetterMessage{values: val}).incrementRetry(); got != 1 {
		t.Fatalf("expected retry 1, got %d", got)
	}
	if got := val[deadLetterRetryField]; got != 1 {
		t.Fatalf("expected retry stored in payload, got %#v", got)
	}
}

func TestDeadLetterRepublishPreservesOriginalMessage(t *testing.T) {
	values := map[string]any{
		"id":                 "message-id",
		"channel":            "default-channel",
		"topic":              "default-topic",
		"orderKey":           "order-42",
		"payload":            `{"customer":"alice","items":[1,2,3]}`,
		"customField":        "must-survive",
		"status":             "received",
		"addTime":            "2026-07-15T16:00:00+08:00",
		"executeTime":        "2026-07-15T16:00:01+08:00",
		"beginTime":          "2026-07-15T16:00:02+08:00",
		"endTime":            "2026-07-15T16:00:03+08:00",
		"retry":              "2",
		"maxLen":             "1000",
		deadLetterRetryField: "1",
	}
	want := make(map[string]any, len(values))
	for key, value := range values {
		want[key] = value
	}

	args := (deadLetterMessage{values: values}).xAddArgs("stream-key", "logic-key")
	if args.Stream != "stream-key" {
		t.Fatalf("expected republish to stream-key, got %q", args.Stream)
	}
	for key, value := range want {
		if key == deadLetterRetryField {
			continue
		}
		if !reflect.DeepEqual(values[key], value) {
			t.Fatalf("field %q changed: got %#v, want %#v", key, values[key], value)
		}
	}
	if got := values[deadLetterRetryField]; got != 2 {
		t.Fatalf("expected dead-letter retry 2, got %#v", got)
	}
	if len(values) != len(want) {
		t.Fatalf("message fields changed: got %#v, want keys from %#v", values, want)
	}
}

func TestDeadLetterMoveToLogicPreservesOriginalMessage(t *testing.T) {
	values := map[string]any{
		"id":                 "message-id",
		"payload":            `{"customer":"alice","items":[1,2,3]}`,
		"customField":        "must-survive",
		"status":             "received",
		"beginTime":          "2026-07-15T16:00:02+08:00",
		"endTime":            "2026-07-15T16:00:03+08:00",
		deadLetterRetryField: 3,
	}
	want := make(map[string]any, len(values))
	for key, value := range values {
		want[key] = value
	}

	args := (deadLetterMessage{values: values}).xAddArgs("stream-key", "logic-key")
	if args.Stream != "logic-key" {
		t.Fatalf("expected move to logic-key, got %q", args.Stream)
	}
	for key, value := range want {
		if key == "status" {
			continue
		}
		if !reflect.DeepEqual(values[key], value) {
			t.Fatalf("field %q changed: got %#v, want %#v", key, values[key], value)
		}
	}
	if got := values["logType"]; got != bstatus.Dlq {
		t.Fatalf("expected DLQ log type, got %#v", got)
	}
	if got := values["status"]; got != bstatus.StatusFailed {
		t.Fatalf("expected failed status, got %#v", got)
	}
	if len(values) != len(want)+1 {
		t.Fatalf("unexpected fields in DLQ message: got %#v, original %#v", values, want)
	}
}
