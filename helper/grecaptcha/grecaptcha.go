package grecaptcha

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	LoginAction     = "LOGIN"
	DefaultMinScore = 0.5
	assessmentURL   = "https://recaptchaenterprise.googleapis.com/v1/projects/%s/assessments"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Config struct {
	Active       bool
	ProjectId    string
	SiteKey      string
	APIKey       string
	MinScore     float64
	HTTPReferrer string
}

type Event struct {
	Token          string `json:"token"`
	SiteKey        string `json:"siteKey"`
	ExpectedAction string `json:"expectedAction"`
	UserAgent      string `json:"userAgent,omitempty"`
	UserIPAddress  string `json:"userIpAddress,omitempty"`
}

type assessmentRequest struct {
	Event Event `json:"event"`
}

type TokenProperties struct {
	Valid         bool   `json:"valid"`
	Action        string `json:"action"`
	InvalidReason string `json:"invalidReason"`
	Hostname      string `json:"hostname"`
}

type RiskAnalysis struct {
	Score     float64  `json:"score"`
	Reasons   []string `json:"reasons"`
	Challenge string   `json:"challenge"`
}

type Assessment struct {
	Name            string          `json:"name"`
	TokenProperties TokenProperties `json:"tokenProperties"`
	RiskAnalysis    RiskAnalysis    `json:"riskAnalysis"`
	Event           Event           `json:"event"`
}

func VerifyLogin(ctx context.Context, cfg Config, token, userAgent, userIP string, client HTTPClient) error {
	return Verify(ctx, cfg, token, LoginAction, userAgent, userIP, client)
}

func Verify(ctx context.Context, cfg Config, token, expectedAction, userAgent, userIP string, client HTTPClient) error {
	if !cfg.Active {
		return nil
	}
	if strings.TrimSpace(cfg.ProjectId) == "" || strings.TrimSpace(cfg.SiteKey) == "" || strings.TrimSpace(cfg.APIKey) == "" {
		return errors.New("google recaptcha is enabled but not fully configured")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("recaptcha token is required")
	}
	if expectedAction == "" {
		expectedAction = LoginAction
	}

	assessment, err := CreateAssessment(ctx, cfg, Event{
		Token:          token,
		SiteKey:        cfg.SiteKey,
		ExpectedAction: expectedAction,
		UserAgent:      userAgent,
		UserIPAddress:  userIP,
	}, client)
	if err != nil {
		return err
	}
	return Interpret(assessment, expectedAction, minScore(cfg.MinScore))
}

func CreateAssessment(ctx context.Context, cfg Config, event Event, client HTTPClient) (*Assessment, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	body, err := json.Marshal(assessmentRequest{Event: event})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf(assessmentURL, cfg.ProjectId)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", cfg.APIKey)
	req.Header.Del("Referer")
	if referrer := strings.TrimSpace(cfg.HTTPReferrer); referrer != "" {
		req.Header.Set("Referer", referrer)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, assessmentError(res.StatusCode, string(raw))
	}
	var assessment Assessment
	if err := json.Unmarshal(raw, &assessment); err != nil {
		return nil, err
	}
	return &assessment, nil
}

func assessmentError(status int, body string) error {
	if status == http.StatusForbidden && strings.Contains(strings.ToLower(body), "referer") {
		return errors.New("recaptcha API key is restricted to HTTP referrers; create a server API key with no application restriction or IP-address restriction, and allow the reCAPTCHA Enterprise API")
	}
	if status == http.StatusForbidden {
		return fmt.Errorf("recaptcha assessment permission denied; check project ID, API key, and that reCAPTCHA Enterprise API is enabled: %s", compactBody(body))
	}
	return fmt.Errorf("recaptcha assessment failed: status=%d body=%s", status, compactBody(body))
}

func compactBody(body string) string {
	body = strings.Join(strings.Fields(body), " ")
	if len(body) > 300 {
		return body[:300]
	}
	return body
}

func Interpret(assessment *Assessment, expectedAction string, minScore float64) error {
	if assessment == nil {
		return errors.New("empty recaptcha assessment")
	}
	if !assessment.TokenProperties.Valid {
		reason := assessment.TokenProperties.InvalidReason
		if reason == "" {
			reason = "invalid token"
		}
		return fmt.Errorf("recaptcha token is invalid: %s", reason)
	}
	action := assessment.TokenProperties.Action
	if action == "" {
		action = assessment.Event.ExpectedAction
	}
	if !strings.EqualFold(action, expectedAction) {
		return fmt.Errorf("recaptcha action mismatch: got %q want %q", action, expectedAction)
	}
	if strings.EqualFold(assessment.RiskAnalysis.Challenge, "FAILED") {
		return errors.New("recaptcha challenge failed")
	}
	if assessment.RiskAnalysis.Score < minScore {
		return fmt.Errorf("recaptcha score too low: %.2f", assessment.RiskAnalysis.Score)
	}
	return nil
}

func minScore(score float64) float64 {
	if score <= 0 {
		return DefaultMinScore
	}
	return score
}
