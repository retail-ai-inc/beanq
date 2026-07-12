package googleAuth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type httpClientFunc func(req *http.Request) (*http.Response, error)

func (f httpClientFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestResponseContextUsesInjectedClient(t *testing.T) {
	auth := NewGoogleOauthConfig("client", "secret", "http://callback")
	auth.WithHTTPClient(httpClientFunc(func(req *http.Request) (*http.Response, error) {
		if req.Context() != context.Background() && req.Context().Err() != nil {
			t.Fatalf("request context should be usable: %v", req.Context().Err())
		}
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", req.Method)
		}
		if got := req.URL.Query().Get("access_token"); got != "token with space" {
			t.Fatalf("unexpected access token query: %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("{\"id\":\"1\",\"email\":\"user@example.com\",\"verified_email\":true}")),
		}, nil
	}))

	user, err := auth.ResponseContext(context.Background(), "token with space")
	if err != nil {
		t.Fatalf("ResponseContext returned error: %v", err)
	}
	if user.Email != "user@example.com" || !user.VerifiedEmail {
		t.Fatalf("unexpected user info: %+v", user)
	}
}

func TestResponseContextReturnsStatusError(t *testing.T) {
	auth := NewGoogleOauthConfig("client", "secret", "http://callback")
	auth.WithHTTPClient(httpClientFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader("unauthorized")),
		}, nil
	}))

	_, err := auth.ResponseContext(context.Background(), "bad-token")
	if err == nil || !strings.Contains(err.Error(), "status=401") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestResponseContextRequiresAccessToken(t *testing.T) {
	auth := NewGoogleOauthConfig("client", "secret", "http://callback")
	_, err := auth.ResponseContext(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "access token is required") {
		t.Fatalf("expected access token error, got %v", err)
	}
}
