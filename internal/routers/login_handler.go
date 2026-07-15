package routers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bjwt"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/googleAuth"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/helper/ui"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/golang-jwt/jwt/v5"
)

type Login struct {
	client redis.UniversalClient
	mgo    *bmongo.BMongo
	prefix string
	ui     ui.Ui
}

func NewLogin(client redis.UniversalClient, mgo *bmongo.BMongo, prefix string, ui ui.Ui) *Login {
	return &Login{client: client, mgo: mgo, prefix: prefix, ui: ui}
}

func (t *Login) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		ExpiredDays int64  `json:"expiredDays"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeBadRequest(w, err)
		return
	}
	username := input.Username
	password := input.Password

	result, cancel := response.Get()
	defer cancel()

	var (
		user = &bmongo.User{
			Account:  "",
			Password: "",
			Type:     "",
			Detail:   "",
			Active:   0,
			RoleId:   "",
			Roles:    []int{},
		}
		err error
	)
	// check Email
	if username != t.ui.Root.UserName {
		if _, err := mail.ParseEmail(username); err != nil {
			result.Code = berror.MissParameterCode
			result.Msg = err.Error()
			_ = result.Json(w, http.StatusBadRequest)
			return
		}
	}

	if username != t.ui.Root.UserName || password != t.ui.Root.Password {
		if t.mgo == nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = "mongo is not configured"
			_ = result.Json(w, http.StatusServiceUnavailable)
			return
		}
		user, err = t.mgo.CheckUser(r.Context(), username, password)
		if err != nil || user == nil {
			result.Code = berror.AuthExpireCode
			result.Msg = "Incorrect username or password"
			_ = result.Json(w, http.StatusUnauthorized)
			return
		}
	}
	expiresAt := t.ui.ExpiresAt
	if input.ExpiredDays > 0 && input.ExpiredDays <= 30 {
		expiresAt = time.Duration(input.ExpiredDays) * 24 * time.Hour
	}

	claim := bjwt.Claim{
		UserName: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    t.ui.Issuer,
			Subject:   t.ui.Subject,
			Audience:  nil,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresAt)),
			NotBefore: nil,
			IssuedAt:  nil,
			ID:        "",
		},
	}

	token, err := bjwt.MakeHsToken(claim, []byte(t.ui.JwtKey))
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}

	client := tool.ClientFac(t.client, t.prefix, "")
	nodeId := client.NodeId(r.Context())

	setAuthCookie(w, r, token, claim.ExpiresAt.Time)
	result.Data = map[string]any{"account": username, "roles": user.Roles, "nodeId": nodeId}
	_ = result.Json(w, http.StatusOK)
}

func (t *Login) Logout(w http.ResponseWriter, r *http.Request) {
	clearAuthCookie(w, r)
	result, release := response.Get()
	defer release()
	_ = result.Json(w, http.StatusOK)
}

func (t *Login) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if t.mgo == nil {
		ReturnHtml(w, "mongo is not configured")
		return
	}

	config, err := t.mgo.ConfigInfo(r.Context())
	if err != nil {
		ReturnHtml(w, err.Error())
		return
	}

	gAuth, err := googleAuth.New(config.Google.ClientId, config.Google.ClientSecret, config.Google.CallBackUrl)
	if err != nil {
		ReturnHtml(w, err.Error())
		return
	}

	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		ReturnHtml(w, "unable to start authentication")
		return
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)
	http.SetCookie(w, &http.Cookie{Name: "beanq_oauth_state", Value: state, Path: "/api/v1/auth/google/callback", MaxAge: 300, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})
	url := gAuth.AuthCodeUrl(state)
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (t *Login) GoogleCallBack(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()
	if t.mgo == nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = "mongo is not configured"
		_ = res.Json(w, http.StatusServiceUnavailable)
		return
	}

	code := r.FormValue("code")
	stateCookie, cookieErr := r.Cookie("beanq_oauth_state")
	if cookieErr != nil || stateCookie.Value == "" || r.FormValue("state") != stateCookie.Value {
		res.Code = berror.AuthExpireCode
		res.Msg = "invalid OAuth state"
		_ = res.Json(w, http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "beanq_oauth_state", Path: "/api/v1/auth/google/callback", MaxAge: -1, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})

	config, err := t.mgo.ConfigInfo(r.Context())
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}

	auth, err := googleAuth.New(config.Google.ClientId, config.Google.ClientSecret, config.Google.CallBackUrl)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	token, err := auth.Exchange(r.Context(), code)

	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}

	userInfo, err := auth.ResponseContext(r.Context(), token.AccessToken)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}

	user, err := t.mgo.CheckGoogleUser(r.Context(), userInfo.Email)

	if err != nil || user == nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}

	claim := bjwt.Claim{
		UserName: userInfo.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    t.ui.Issuer,
			Subject:   t.ui.Subject,
			Audience:  nil,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.ui.ExpiresAt)),
			NotBefore: nil,
			IssuedAt:  nil,
			ID:        "",
		},
	}
	jwtToken, err := bjwt.MakeHsToken(claim, []byte(t.ui.JwtKey))
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	setAuthCookie(w, r, jwtToken, claim.ExpiresAt.Time)
	callbackURL, err := url.Parse(config.Google.CallBackUrl)
	if err != nil || callbackURL.Scheme == "" || callbackURL.Host == "" {
		res.Code = berror.InternalServerErrorCode
		res.Msg = "invalid configured Google callback URL"
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	redirectURL := fmt.Sprintf("%s://%s/#/login?authenticated=1", callbackURL.Scheme, callbackURL.Host)

	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	w.Header().Set("Location", redirectURL)
	w.WriteHeader(http.StatusFound)
}

func (t *Login) LoginAllowGoogle(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()
	if t.mgo == nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = "mongo is not configured"
		_ = res.Json(w, http.StatusServiceUnavailable)
		return
	}

	config, err := t.mgo.ConfigInfo(r.Context())

	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"google":          config.Google,
		"googleReCAPTCHA": config.GoogleReCAPTCHA,
	}
	res.Data = data
	_ = res.Json(w, http.StatusOK)
}

func (t *Login) TestNotify(w http.ResponseWriter, r *http.Request) {
	result, cancel := response.Get()
	defer cancel()

	var data = struct {
		SMTP     capture.SMTP     `json:"smtp"`
		SendGrid capture.SendGrid `json:"sendGrid"`
		Tools    []capture.Then   `json:"tools"`
		Slack    capture.Slack    `json:"slack"`
	}{}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		result.Code = berror.MissParameterCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusBadRequest)
		return
	}

	capture.System.When(&capture.Config{
		Email: capture.Email{
			SMTP:     data.SMTP,
			SendGrid: data.SendGrid,
		},
		Slack: data.Slack,
		Rule: capture.Rule{
			When: []capture.When{{Key: string(capture.System), Value: string(capture.System)}},
			If:   nil,
			Then: data.Tools,
		},
	}).Then(errors.New("test"))

	_ = result.Json(w, http.StatusOK)
}
