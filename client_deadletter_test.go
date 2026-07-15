package beanq

import (
	"context"
	"errors"
	"testing"
	"time"

	public "github.com/retail-ai-inc/beanq/v4/internal"
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
	if got := msg.ToMap()["deadLetterRetry"]; got != 2 {
		t.Fatalf("expected deadLetterRetry in payload map, got %#v", got)
	}
}

func TestNewClientFromConfigDefaultsDeadLetterRetry(t *testing.T) {
	client := newClientFromConfig(&BeanqConfig{})
	if client.DeadLetterRetry != 0 {
		t.Fatalf("expected default deadletter retry 0, got %d", client.DeadLetterRetry)
	}
}

type invokeQueue struct {
	callback public.CallbackWithRetry
}

func (q *invokeQueue) Enqueue(context.Context, map[string]any) error { return nil }

func (q *invokeQueue) Consume(_ context.Context, _ btype.MoodType, _, _ string, callback public.CallbackWithRetry) error {
	q.callback = callback
	return nil
}

func (q *invokeQueue) WaitingAck(context.Context, string, string, string) (map[string]string, error) {
	return nil, nil
}

func TestHandlerInvokeCallsConsumerOnceAndMarksIgnoredError(t *testing.T) {
	wantErr := errors.New("ignored")
	calls := 0
	handler := &Handler{
		do: public.NewCallbackWithRetry(func(context.Context, map[string]any, ...int) (int, error) {
			calls++
			return 7, wantErr
		}, nil),
		retryCond: map[string]struct{}{retryConditionKey(wantErr): {}},
	}
	queue := &invokeQueue{}
	handler.Invoke(context.Background(), queue)

	attempt, err := queue.callback.Handle(context.Background(), nil, 0)
	if calls != 1 {
		t.Fatalf("expected one consumer call per Handle, got %d", calls)
	}
	if attempt != 7 {
		t.Fatalf("expected consumer result 7, got %d", attempt)
	}
	stopped, ok := errors.AsType[interface {
		error
		BeanqRetryStopped() error
	}](err)
	if !ok || stopped.BeanqRetryStopped() != wantErr {
		t.Fatalf("expected retry-stop wrapper around original error, got %v", err)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapper to support errors.Is for %v, got %v", wantErr, err)
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
