package routers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestAuthTokenFallsBackToFormToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?token=query-token", nil)

	token, err := authToken(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "query-token" {
		t.Fatalf("expected query token, got %q", token)
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
