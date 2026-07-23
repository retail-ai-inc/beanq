package routers

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/ui"
)

type deadlineRecorder struct {
	*httptest.ResponseRecorder
	deadline time.Time
	called   bool
}

func TestHeaderRuleAllowsBootstrapDataImages(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	HeaderRule()(func(http.ResponseWriter, *http.Request) {})(res, req)

	policy := res.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "img-src 'self' data:;") {
		t.Fatalf("CSP does not allow Bootstrap data images: %q", policy)
	}
	if strings.Contains(policy, "script-src 'self' data:") {
		t.Fatalf("CSP unexpectedly allows data scripts: %q", policy)
	}
}

func (w *deadlineRecorder) SetWriteDeadline(deadline time.Time) error {
	w.deadline = deadline
	w.called = true
	return nil
}

func TestAuthTokenPrefersAuthorizationBearer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")

	token, err := authToken(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "header-token" {
		t.Fatalf("expected header token, got %q", token)
	}
}

func TestAuthTokenRejectsQueryToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?token=query-token", nil)

	if _, err := authToken(req); err == nil {
		t.Fatal("expected query token to be rejected")
	}
}

func TestAuthTokenPrefersSessionCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: "cookie-token"})
	req.Header.Set("Authorization", "Bearer header-token")
	token, err := authToken(req)
	if err != nil || token != "cookie-token" {
		t.Fatalf("expected cookie token, got %q err=%v", token, err)
	}
}

func TestPermissionComesFromRoutePattern(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/123", nil)
	req.Pattern = "DELETE /api/v1/users/{id}"
	req.Header.Set("X-Role-Id", "1")
	if got := permissionForRequest(req); got != 24 {
		t.Fatalf("expected server permission 24, got %d", got)
	}
}

func TestClearSSEWriteDeadline(t *testing.T) {
	w := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder(), deadline: time.Now()}
	if err := clearSSEWriteDeadline(w); err != nil {
		t.Fatalf("clear write deadline: %v", err)
	}
	if !w.called {
		t.Fatal("expected ResponseController to set the write deadline")
	}
	if !w.deadline.IsZero() {
		t.Fatalf("expected zero deadline, got %v", w.deadline)
	}
}

func TestClearedSSEDeadlineOutlivesServerWriteTimeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := clearSSEWriteDeadline(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		flusher := w.(http.Flusher)
		_, _ = io.WriteString(w, "data:first\n\n")
		flusher.Flush()
		time.Sleep(150 * time.Millisecond)
		_, _ = io.WriteString(w, "data:second\n\n")
	})
	server := httptest.NewUnstartedServer(handler)
	server.Config.WriteTimeout = 50 * time.Millisecond
	server.Start()
	defer server.Close()

	response, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatalf("open SSE response: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read SSE response: %v", err)
	}
	if !strings.Contains(string(body), "data:second") {
		t.Fatalf("SSE response ended at the server write timeout: %q", body)
	}
}

func TestAuthTokenRejectsMalformedAuthorizationHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic abc")

	if _, err := authToken(req); err == nil {
		t.Fatal("expected malformed authorization header error")
	}
}

func TestWorkflowListReturnsUnavailableWhenStorageMissing(t *testing.T) {
	handler := NewWorkFlow(nil)
	req := httptest.NewRequest(http.MethodGet, "/workflow/list", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestWorkflowDeleteRequiresID(t *testing.T) {
	handler := NewWorkFlow(nil)
	req := httptest.NewRequest(http.MethodPost, "/workflow/delete", nil)
	w := httptest.NewRecorder()

	handler.Delete(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected missing storage to be reported before id validation, got %d", w.Code)
	}
}

func TestLegacyAllowGoogleRouteIsRegistered(t *testing.T) {
	assets := fstest.MapFS{"ui/index.html": &fstest.MapFile{Data: []byte("ok")}}
	modified := map[string]time.Time{"/index.html": time.Now()}
	router := RouterList(fs.FS(assets), modified, nil, nil, nil, "beanq", ui.Ui{})
	req := httptest.NewRequest(http.MethodGet, "/login/allowGoogle", nil)
	w := httptest.NewRecorder()

	router.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected registered route to report missing mongo with %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestStaticAssetCachingAndIsolation(t *testing.T) {
	modifiedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	assets := fstest.MapFS{
		"ui/index.html":    &fstest.MapFile{Data: []byte("index")},
		"ui/app.js":        &fstest.MapFile{Data: []byte("console.log(1)")},
		"ui2/index.html":   &fstest.MapFile{Data: []byte("wrong root")},
		"other/index.html": &fstest.MapFile{Data: []byte("other")},
	}
	router := RouterList(fs.FS(assets), map[string]time.Time{
		"/index.html": modifiedAt,
		"/app.js":     modifiedAt,
	}, nil, nil, nil, "beanq", ui.Ui{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	router.Mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK || res.Body.String() != "index" {
		t.Fatalf("unexpected index response: status=%d body=%q", res.Code, res.Body.String())
	}
	etag := res.Header().Get("ETag")
	if etag == "" || res.Header().Get("Last-Modified") != modifiedAt.Format(http.TimeFormat) {
		t.Fatalf("missing standard cache headers: %#v", res.Header())
	}
	if got := res.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-None-Match", etag)
	res = httptest.NewRecorder()
	router.Mux.ServeHTTP(res, req)
	if res.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match status = %d, want %d", res.Code, http.StatusNotModified)
	}

	req = httptest.NewRequest(http.MethodGet, "/ui2/index.html", nil)
	res = httptest.NewRecorder()
	router.Mux.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("ui2 path status = %d, want %d", res.Code, http.StatusNotFound)
	}
}
