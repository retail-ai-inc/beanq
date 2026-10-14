package grecaptcha

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestVerifySkipsWhenInactive(t *testing.T) {
	err := VerifyLogin(context.Background(), Config{Active: false}, "", "", "", nil)
	if err != nil {
		t.Fatalf("expected skip, got %v", err)
	}
}

func TestVerifyRequiresTokenWhenActive(t *testing.T) {
	err := VerifyLogin(context.Background(), Config{Active: true, ProjectId: "p", SiteKey: "s", APIKey: "k"}, "", "", "", nil)
	if err == nil || !strings.Contains(err.Error(), "token is required") {
		t.Fatalf("got %v", err)
	}
}

func TestVerifyAcceptsValidAssessment(t *testing.T) {
	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "/projects/demo/assessments") {
			t.Fatalf("unexpected url %s", req.URL.String())
		}
		body := `{"tokenProperties":{"valid":true,"action":"LOGIN"},"riskAnalysis":{"score":0.9,"challenge":"PASS"}}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(body)), Header: make(http.Header)}, nil
	})
	err := VerifyLogin(context.Background(), Config{Active: true, ProjectId: "demo", SiteKey: "site", APIKey: "key"}, "token", "ua", "1.1.1.1", client)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInterpretRejectsLowScoreAndFailedChallenge(t *testing.T) {
	err := Interpret(&Assessment{TokenProperties: TokenProperties{Valid: true, Action: "LOGIN"}, RiskAnalysis: RiskAnalysis{Score: 0.1}}, LoginAction, DefaultMinScore)
	if err == nil {
		t.Fatal("expected low score error")
	}
	err = Interpret(&Assessment{TokenProperties: TokenProperties{Valid: true, Action: "LOGIN"}, RiskAnalysis: RiskAnalysis{Score: 0.9, Challenge: "FAILED"}}, LoginAction, DefaultMinScore)
	if err == nil {
		t.Fatal("expected failed challenge")
	}
}

func TestAssessmentMapsRefererForbidden(t *testing.T) {
	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.RawQuery != "" {
			t.Fatalf("api key must not be in query, got %s", req.URL.String())
		}
		if req.Header.Get("X-Goog-Api-Key") != "key" {
			t.Fatal("missing api key header")
		}
		body := `{ "error": { "code": 403, "message": "Requests from this referer are blocked.", "status": "PERMISSION_DENIED" } }`
		return &http.Response{StatusCode: 403, Body: io.NopCloser(bytes.NewBufferString(body)), Header: make(http.Header)}, nil
	})
	err := VerifyLogin(context.Background(), Config{Active: true, ProjectId: "demo", SiteKey: "site", APIKey: "key"}, "token", "ua", "1.1.1.1", client)
	if err == nil || !strings.Contains(err.Error(), "HTTP referrers") {
		t.Fatalf("got %v", err)
	}
}
