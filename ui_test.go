package beanq

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"
)

func TestUIListenAddr(t *testing.T) {
	tests := []struct {
		name string
		port string
		want string
	}{
		{name: "plain port", port: "8080", want: ":8080"},
		{name: "prefixed port", port: ":8080", want: ":8080"},
		{name: "empty port keeps previous behavior", port: "", want: ":"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uiListenAddr(tt.port); got != tt.want {
				t.Fatalf("uiListenAddr(%q) = %q, want %q", tt.port, got, tt.want)
			}
		})
	}
}

func TestUICollectionName(t *testing.T) {
	cfg := &Mongo{Collections: map[string]Collection{
		"workflow": {Name: "workflow_custom"},
		"empty":    {Name: ""},
	}}

	if got := uiCollectionName(cfg, "workflow", "workflow_records"); got != "workflow_custom" {
		t.Fatalf("expected configured workflow collection, got %q", got)
	}
	if got := uiCollectionName(cfg, "missing", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback collection, got %q", got)
	}
	if got := uiCollectionName(cfg, "empty", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback for empty collection name, got %q", got)
	}
}

func TestStaticFileInfo(t *testing.T) {
	files, err := StaticFileInfo(fstest.MapFS{
		"ui/index.html":        &fstest.MapFile{Data: []byte("index")},
		"ui/static/app.js":     &fstest.MapFile{Data: []byte("app")},
		"other/ignored.txt":    &fstest.MapFile{Data: []byte("ignored")},
		"ui/static/nested.css": &fstest.MapFile{Data: []byte("css")},
	})
	if err != nil {
		t.Fatalf("StaticFileInfo returned error: %v", err)
	}

	for _, name := range []string{"/index.html", "/static/app.js", "/static/nested.css"} {
		if _, ok := files[name]; !ok {
			t.Fatalf("expected %q in static file info: %#v", name, files)
		}
	}
	if _, ok := files["other/ignored.txt"]; ok {
		t.Fatalf("did not expect non-ui file in static file info: %#v", files)
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
