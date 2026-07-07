package googleAuth

import (
	"context"

	"golang.org/x/oauth2"
)

// Provider is the test-friendly boundary for Google OAuth integrations.
type Provider interface {
	AuthCodeUrl(state string, opts ...oauth2.AuthCodeOption) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	Response(accessToken string) (*UserInfo, error)
	ResponseContext(ctx context.Context, accessToken string) (*UserInfo, error)
}

var _ Provider = (*GoogleOauthConfig)(nil)
