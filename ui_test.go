package beanq

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
)

func TestUIListenAddr(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		want    string
		wantErr bool
	}{
		{name: "plain port", port: "8080", want: ":8080"},
		{name: "prefixed port", port: ":8080", want: ":8080"},
		{name: "trim whitespace", port: " 8080 ", want: ":8080"},
		{name: "empty port", port: "", wantErr: true},
		{name: "non numeric port", port: "http", wantErr: true},
		{name: "zero port", port: "0", wantErr: true},
		{name: "negative port", port: "-1", wantErr: true},
		{name: "port above maximum", port: "65536", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := uiListenAddr(tt.port)
			if (err != nil) != tt.wantErr {
				t.Fatalf("uiListenAddr(%q) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("uiListenAddr(%q) = %q, want %q", tt.port, got, tt.want)
			}
		})
	}
}

func TestMongoCollectionName(t *testing.T) {
	cfg := &Mongo{Collections: map[string]Collection{
		"workflow": {Name: "workflow_custom"},
		"empty":    {Name: ""},
	}}

	if got := cfg.collectionName("workflow", "workflow_records"); got != "workflow_custom" {
		t.Fatalf("expected configured workflow collection, got %q", got)
	}
	if got := cfg.collectionName("missing", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback collection, got %q", got)
	}
	if got := cfg.collectionName("empty", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback for empty collection name, got %q", got)
	}
}

func TestStaticFileInfo(t *testing.T) {
	files, err := staticFileInfo(fstest.MapFS{
		"ui/index.html":        &fstest.MapFile{Data: []byte("index")},
		"ui/static/app.js":     &fstest.MapFile{Data: []byte("app")},
		"ui2/ignored.txt":      &fstest.MapFile{Data: []byte("ignored")},
		"other/ignored.txt":    &fstest.MapFile{Data: []byte("ignored")},
		"ui/static/nested.css": &fstest.MapFile{Data: []byte("css")},
	})
	if err != nil {
		t.Fatalf("staticFileInfo returned error: %v", err)
	}

	for _, name := range []string{"/index.html", "/static/app.js", "/static/nested.css"} {
		if _, ok := files[name]; !ok {
			t.Fatalf("expected %q in static file info: %#v", name, files)
		}
	}
	for _, name := range []string{"/ignored.txt", "other/ignored.txt"} {
		if _, ok := files[name]; ok {
			t.Fatalf("did not expect %q in static file info: %#v", name, files)
		}
	}
}

func TestStaticFileInfoRequiresUIRoot(t *testing.T) {
	if _, err := staticFileInfo(fstest.MapFS{
		"other/file.txt": &fstest.MapFile{Data: []byte("ignored")},
	}); err == nil {
		t.Fatal("expected missing ui root to return an error")
	}
}

type fakeAdminReporter struct {
	calls      atomic.Int32
	concurrent atomic.Int32
	maxActive  atomic.Int32
	called     chan struct{}
}

func (f *fakeAdminReporter) HostName(context.Context) error { return nil }

func (f *fakeAdminReporter) QueueMessage(ctx context.Context) error {
	f.calls.Add(1)
	active := f.concurrent.Add(1)
	defer f.concurrent.Add(-1)
	for {
		maximum := f.maxActive.Load()
		if active <= maximum || f.maxActive.CompareAndSwap(maximum, active) {
			break
		}
	}
	select {
	case f.called <- struct{}{}:
	default:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Millisecond):
		return nil
	}
}

func TestRunUIQueueReporter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reporter := &fakeAdminReporter{called: make(chan struct{}, 4)}
	done := make(chan struct{})
	go func() {
		runUIQueueReporter(ctx, reporter, time.Millisecond)
		close(done)
	}()

	select {
	case <-reporter.called:
	case <-time.After(time.Second):
		t.Fatal("reporter did not run immediately")
	}
	select {
	case <-reporter.called:
	case <-time.After(time.Second):
		t.Fatal("reporter did not run periodically")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("reporter did not stop after cancellation")
	}
	if got := reporter.maxActive.Load(); got != 1 {
		t.Fatalf("reporter had %d concurrent calls, want 1", got)
	}
}

func TestNewUIServerTimeouts(t *testing.T) {
	server := newUIServer(":9090", http.NewServeMux())
	if server.ReadHeaderTimeout != uiReadHeaderTimeout || server.ReadTimeout != uiReadTimeout ||
		server.WriteTimeout != uiWriteTimeout || server.IdleTimeout != uiIdleTimeout {
		t.Fatalf("unexpected UI server timeouts: %#v", server)
	}
}

func TestRunUIServerReturnsListenError(t *testing.T) {
	err := runUIServer(context.Background(), newUIServer("127.0.0.1:-1", http.NewServeMux()), time.Second)
	if err == nil || !strings.Contains(err.Error(), "serve UI") {
		t.Fatalf("expected wrapped listen error, got %v", err)
	}
}

func TestRunUIServerStopsWithCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runUIServer(ctx, newUIServer("127.0.0.1:0", http.NewServeMux()), time.Second); err != nil {
		t.Fatalf("runUIServer returned error while stopping: %v", err)
	}
}

func TestIsUIShutdownTimeout(t *testing.T) {
	if !isUIShutdownTimeout(context.DeadlineExceeded) {
		t.Fatal("expected context deadline exceeded to be treated as shutdown timeout")
	}
	if !isUIShutdownTimeout(context.Canceled) {
		t.Fatal("expected context canceled to be treated as shutdown timeout")
	}
	if isUIShutdownTimeout(errors.New("boom")) {
		t.Fatal("did not expect unrelated errors to be treated as shutdown timeout")
	}
}
