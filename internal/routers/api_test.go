package routers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakePublisher struct{ called bool }

func (f *fakePublisher) Publish(context.Context, map[string]any) error { f.called = true; return nil }

func TestParsePageBoundaries(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=1&pageSize=101", nil)
	if _, err := parsePage(req); err == nil {
		t.Fatal("expected oversized pageSize error")
	}
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	page, err := parsePage(req)
	if err != nil || page.Page != 1 || page.PageSize != 10 {
		t.Fatalf("unexpected defaults: %+v %v", page, err)
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"known":1,"extra":2}`))
	w := httptest.NewRecorder()
	var input struct {
		Known int `json:"known"`
	}
	if err := decodeJSON(w, req, &input); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestPublishRetryRejectsUnsupportedType(t *testing.T) {
	err := publishRetry(context.Background(), map[string]any{"moodType": "sequential"}, func(string) publishQueue { return nil })
	if err == nil {
		t.Fatal("expected sequential retry to fail")
	}
}

func TestPublishRetryUsesFactory(t *testing.T) {
	publisher := &fakePublisher{}
	err := publishRetry(context.Background(), map[string]any{"moodType": "normal"}, func(string) publishQueue { return publisher })
	if err != nil || !publisher.called {
		t.Fatalf("publisher not called: %v", err)
	}
}

func TestCompareStreamIDUsesNumericOrder(t *testing.T) {
	if compareStreamID("10-2", "9-100") <= 0 {
		t.Fatal("expected 10-2 after 9-100")
	}
	if compareStreamID("10-2", "10-11") >= 0 {
		t.Fatal("expected sequence 2 before 11")
	}
}

func TestStreamEventsEmitsImmediatelyAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/stream", nil).WithContext(ctx)
	res := httptest.NewRecorder()
	calls := 0
	err := streamEvents(res, req, time.Millisecond, func() error {
		calls++
		if calls == 2 {
			cancel()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("streamEvents returned error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("streamEvents emitted %d times, want 2", calls)
	}
	if got := res.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q", got)
	}
}
