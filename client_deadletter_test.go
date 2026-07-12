package beanq

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/retail-ai-inc/beanq/v4/internal/btype"
)

func TestBuildMessageIncludesDeadLetterRetry(t *testing.T) {
	client := &Client{
		Topic:           "topic",
		Channel:         "channel",
		MaxLen:          10,
		Retry:           1,
		DeadLetterRetry: 2,
		TimeToRun:       time.Second,
	}
	bq := &BQClient{client: client, priority: 5}

	msg := bq.buildMessage(&Publish{
		payload:     []byte("payload"),
		moodType:    btype.NORMAL,
		executeTime: time.Now(),
	})

	if msg.DeadLetterRetry != 2 {
		t.Fatalf("expected message deadletter retry 2, got %d", msg.DeadLetterRetry)
	}
	if got := msg.ToMap()["deadletterRetry"]; got != 2 {
		t.Fatalf("expected deadletterRetry in payload map, got %#v", got)
	}
}

func TestNewClientFromConfigDefaultsDeadLetterRetry(t *testing.T) {
	client := newClientFromConfig(&BeanqConfig{})
	if client.DeadLetterRetry != 0 {
		t.Fatalf("expected default deadletter retry 0, got %d", client.DeadLetterRetry)
	}
}

func TestConsumerCallbackUsesSubscribeErrorHook(t *testing.T) {
	want := errors.New("handler failed")
	var got error
	handler := DefaultHandle{
		DoHandle: func(ctx context.Context, message *Message) error {
			return want
		},
		DoCancel: func(ctx context.Context, message *Message) error {
			return nil
		},
		DoError: func(ctx context.Context, err error) {
			got = err
		},
	}

	callback := consumerCallback(handler)
	_, err := callback.Handle(context.Background(), map[string]any{"payload": "payload"})
	if !errors.Is(err, want) {
		t.Fatalf("expected handle error %v, got %v", want, err)
	}
	callback.Error(context.Background(), err)
	if !errors.Is(got, want) {
		t.Fatalf("expected DoError to receive %v, got %v", want, got)
	}
}
