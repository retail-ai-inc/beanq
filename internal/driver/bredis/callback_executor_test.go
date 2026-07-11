package bredis

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	public "github.com/retail-ai-inc/beanq/v4/internal"
)

type executorHandler struct {
	mu      sync.Mutex
	calls   int
	retries []int
	handle  func(context.Context, map[string]any, int) error
	errors  []error
}

func (h *executorHandler) Handle(ctx context.Context, data map[string]any, retry ...int) (int, error) {
	h.mu.Lock()
	h.calls++
	attempt := 0
	if len(retry) > 0 {
		attempt = retry[0]
	}
	h.retries = append(h.retries, attempt)
	h.mu.Unlock()
	if h.handle == nil {
		return 0, nil
	}
	return 0, h.handle(ctx, data, attempt)
}

func (h *executorHandler) Error(_ context.Context, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errors = append(h.errors, err)
}

func executorJob(data map[string]any) public.Stream {
	return public.Stream{Data: data, Id: "1", Channel: "channel", Stream: "stream"}
}

func TestExecuteMessageSuccess(t *testing.T) {
	data := map[string]any{"payload": "original", "retry": 0, "timeToRun": time.Second}
	handler := &executorHandler{handle: func(_ context.Context, copied map[string]any, _ int) error {
		copied["payload"] = "changed"
		return nil
	}}

	result, ok := executeMessage(context.Background(), executorJob(data), handler, nil)
	if !ok {
		t.Fatal("expected completed result")
	}
	if got := data["payload"]; got != "original" {
		t.Fatalf("handler mutated original map: %#v", got)
	}
	if got := result.Data["status"]; got != bstatus.StatusSuccess {
		t.Fatalf("expected success status, got %#v", got)
	}
	assertExecutionMetadata(t, result.Data)
	if len(handler.errors) != 0 {
		t.Fatalf("unexpected errors: %v", handler.errors)
	}
}

func TestExecuteMessageErrorRetries(t *testing.T) {
	wantErr := errors.New("handler failed")
	handler := &executorHandler{handle: func(context.Context, map[string]any, int) error { return wantErr }}
	data := map[string]any{"retry": 1, "timeToRun": 3 * time.Second}

	result, ok := executeMessage(context.Background(), executorJob(data), handler, nil)
	if !ok {
		t.Fatal("expected completed result")
	}
	if handler.calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", handler.calls)
	}
	if len(handler.errors) != 1 || !errors.Is(handler.errors[0], wantErr) {
		t.Fatalf("expected Error callback with handler error, got %v", handler.errors)
	}
	if got := result.Data["retry"]; got != 1 {
		t.Fatalf("expected retry index 1, got %#v", got)
	}
	if got := result.Data["status"]; got != bstatus.StatusFailed {
		t.Fatalf("expected failed status, got %#v", got)
	}
	if got := result.Data["level"]; got != bstatus.ErrLevel {
		t.Fatalf("expected error level, got %#v", got)
	}
	if got := result.Data["info"]; got != wantErr.Error() {
		t.Fatalf("expected error info %q, got %#v", wantErr, got)
	}
}

func TestExecuteMessageRecoversPanic(t *testing.T) {
	handler := &executorHandler{handle: func(context.Context, map[string]any, int) error {
		panic("boom")
	}}

	result, ok := executeMessage(context.Background(), executorJob(map[string]any{"timeToRun": time.Second}), handler, nil)
	if !ok {
		t.Fatal("expected completed result")
	}
	if got := result.Data["status"]; got != bstatus.StatusFailed {
		t.Fatalf("expected failed status, got %#v", got)
	}
	if len(handler.errors) != 1 || !strings.Contains(handler.errors[0].Error(), "[panic recover]: boom") {
		t.Fatalf("expected recovered panic in Error callback, got %v", handler.errors)
	}
}

func TestWorkerContextCancelDoesNotPublishResultOrCallError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	handler := &executorHandler{handle: func(ctx context.Context, _ map[string]any, _ int) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}}
	jobs := make(chan public.Stream, 1)
	results := make(chan public.Stream)
	var wg sync.WaitGroup
	wg.Add(1)
	go worker(ctx, jobs, results, handler, &wg, nil)
	jobs <- executorJob(map[string]any{"timeToRun": time.Minute})
	close(jobs)

	<-started
	cancel()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker blocked after context cancellation")
	}
	select {
	case result := <-results:
		t.Fatalf("unexpected result after cancellation: %#v", result)
	default:
	}
	if len(handler.errors) != 0 {
		t.Fatalf("Error callback must not run after parent cancellation: %v", handler.errors)
	}
}

func TestExecuteMessageInvalidTimeToRunLimitDoesNotPanic(t *testing.T) {
	result, ok := executeMessage(context.Background(), executorJob(map[string]any{
		"timeToRun":      time.Second,
		"timeToRunLimit": []time.Duration{time.Second},
	}), &executorHandler{}, nil)
	if !ok || result.Data["status"] != bstatus.StatusSuccess {
		t.Fatalf("expected successful execution, got ok=%v data=%#v", ok, result.Data)
	}
}

func assertExecutionMetadata(t *testing.T, data map[string]any) {
	t.Helper()
	for _, key := range []string{"beginTime", "endTime", "runTime", "hostName", "retry"} {
		if _, ok := data[key]; !ok {
			t.Errorf("expected %s metadata in %#v", key, data)
		}
	}
}
