package googleAuth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// user info
type UserInfo struct {
	Id            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
	VerifiedEmail bool   `json:"verified_email"`
}

type GoogleOauthConfig struct {
	config     *oauth2.Config
	httpClient HTTPClient
}

// HTTPClient is the minimal boundary used to fetch Google user info.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func New(clientId, clientSecret, redirectUrl string) (*GoogleOauthConfig, error) {
	if clientId == "" || clientSecret == "" || redirectUrl == "" {
		return nil, fmt.Errorf("[google auth]error:%w", errors.New("missing parameter"))
	}
	return NewGoogleOauthConfig(clientId, clientSecret, redirectUrl), nil
}
func NewGoogleOauthConfig(clientId, clientSecret, redirectUrl string) *GoogleOauthConfig {
	//endpoint := oauth2.Endpoint{
	//	AuthURL:   AuthUrl,
	//	TokenURL:  TokenUrl,
	//	AuthStyle: oauth2.AuthStyleInParams,
	//}
	endpoint := google.Endpoint
	return &GoogleOauthConfig{
		config: &oauth2.Config{
			ClientID:     clientId,
			ClientSecret: clientSecret,
			Endpoint:     endpoint,
			RedirectURL:  redirectUrl,
			//Scopes: []string{"https://www.googleapis.com/auth/userinfo.profile","https://www.googleapis.com/auth/userinfo.email"},
			Scopes: []string{"profile", "email"},
		},
		httpClient: http.DefaultClient,
	}
}

func (t *GoogleOauthConfig) WithHTTPClient(client HTTPClient) *GoogleOauthConfig {
	if client != nil {
		t.httpClient = client
	}
	return t
}
func (t *GoogleOauthConfig) AuthCodeUrl(state string, opts ...oauth2.AuthCodeOption) (url string) {
	url = t.config.AuthCodeURL(state, opts...)
	return
}
func (t *GoogleOauthConfig) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return t.config.Exchange(ctx, code)
}
func (t *GoogleOauthConfig) Response(accessToken string) (*UserInfo, error) {
	return t.ResponseContext(context.Background(), accessToken)
}

func (t *GoogleOauthConfig) ResponseContext(ctx context.Context, accessToken string) (*UserInfo, error) {
	if accessToken == "" {
		return nil, errors.New("access token is required")
	}
	if t.httpClient == nil {
		t.httpClient = http.DefaultClient
	}
	endpoint := "https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + url.QueryEscape(accessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	res, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	bodys, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("google userinfo request failed: status=%d body=%s", res.StatusCode, string(bodys))
	}
	var userInfo UserInfo
	err = json.Unmarshal(bodys, &userInfo)
	if err != nil {
		return nil, err
	}
	return &userInfo, nil
}
