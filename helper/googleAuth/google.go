package googleAuth

import (
	"context"
	"encoding/json"
	"github.com/spf13/viper"
	"io"
	"net/http"

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
	config *oauth2.Config
}

func New() *GoogleOauthConfig {

	clientId := viper.GetString("googleAuth.clientId")
	clientSecret := viper.GetString("googleAuth.clientSecret")
	redirectUrl := viper.GetString("googleAuth.callbackUrl")

	return NewGoogleOauthConfig(clientId, clientSecret, redirectUrl)
}
func NewGoogleOauthConfig(clientId, clientSecret, redirectUrl string) *GoogleOauthConfig {
	//endpoint := oauth2.Endpoint{
	//	AuthURL:   AuthUrl,
	//	TokenURL:  TokenUrl,
	//	AuthStyle: oauth2.AuthStyleInParams,
	//}
	endpoint := google.Endpoint
	return &GoogleOauthConfig{
		&oauth2.Config{
			ClientID:     clientId,
			ClientSecret: clientSecret,
			Endpoint:     endpoint,
			RedirectURL:  redirectUrl,
			//Scopes: []string{"https://www.googleapis.com/auth/userinfo.profile","https://www.googleapis.com/auth/userinfo.email"},
			Scopes: []string{"profile", "email"},
		},
	}
}
func (t *GoogleOauthConfig) AuthCodeUrl(state string, opts ...oauth2.AuthCodeOption) (url string) {
	url = t.config.AuthCodeURL(state, opts...)
	return
}
func (t *GoogleOauthConfig) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return t.config.Exchange(ctx, code)
}
func (t *GoogleOauthConfig) Response(accessToken string) (*UserInfo, error) {
	res, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	bodys, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var userInfo UserInfo
	err = json.Unmarshal(bodys, &userInfo)
	if err != nil {
		return nil, err
	}
	return &userInfo, nil
}
