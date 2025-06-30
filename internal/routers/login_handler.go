package routers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bjwt"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/googleAuth"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	"github.com/retail-ai-inc/beanq/v4/helper/ui"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/cast"

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

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	expiredTime := r.PostFormValue("expiredTime")

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
		user, err = t.mgo.CheckUser(r.Context(), username, password)
		if err != nil || user == nil {
			result.Code = berror.AuthExpireCode
			result.Msg = "Incorrect username or password"
			_ = result.Json(w, http.StatusUnauthorized)
			return
		}
	}
	expiresAt := t.ui.ExpiresAt
	if cast.ToInt64(expiredTime) > 0 {
		expiresAt = time.Duration(cast.ToInt64(expiredTime)) * 24 * time.Hour
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

	result.Data = map[string]any{"token": token, "roles": user.Roles, "nodeId": nodeId}
	_ = result.Json(w, http.StatusOK)
}

func (t *Login) GoogleLogin(w http.ResponseWriter, r *http.Request) {

	config, err := t.client.HGetAll(r.Context(), strings.Join([]string{t.prefix, "config"}, ":")).Result()
	if err != nil {
		ReturnHtml(w, err.Error())
		return
	}

	var google capture.GoogleCredential
	if v, ok := config["google"]; ok {
		if err := json.NewDecoder(strings.NewReader(v)).Decode(&google); err != nil {
			ReturnHtml(w, err.Error())
			return
		}
	}

	gAuth, err := googleAuth.New(google.ClientId, google.ClientSecret, google.CallBackUrl)
	if err != nil {
		ReturnHtml(w, err.Error())
		return
	}

	state := time.Now().String()
	url := gAuth.AuthCodeUrl(state)
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (t *Login) GoogleCallBack(w http.ResponseWriter, r *http.Request) {

	res, cancel := response.Get()
	defer cancel()

	code := r.FormValue("code")
	clientId := t.ui.GoogleAuth.ClientId
	clientSecret := t.ui.GoogleAuth.ClientSecret
	callbackUrl := t.ui.GoogleAuth.CallbackUrl
	auth, err := googleAuth.New(clientId, clientSecret, callbackUrl)
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

	userInfo, err := auth.Response(token.AccessToken)
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
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
		if r.TLS != nil {
			proto = "https"
		}
	}
	url := fmt.Sprintf("%s://%s/#/login?token=%s", proto, r.Host, jwtToken)

	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusFound)
}

func (t *Login) LoginAllowGoogle(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	result, err := t.client.HGet(r.Context(), strings.Join([]string{t.prefix, "config"}, ":"), "google").Result()
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
	}

	var data capture.GoogleCredential
	if err := json.NewDecoder(strings.NewReader(result)).Decode(&data); err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
	}

	b := false
	if data.ClientId != "" && data.ClientSecret != "" && data.CallBackUrl != "" && data.Scheme != "" {
		b = true
	}
	res.Data = b
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
		SMTP:     data.SMTP,
		Slack:    data.Slack,
		SendGrid: data.SendGrid,
		Rule: capture.Rule{
			When: []capture.When{{Key: string(capture.System), Value: string(capture.System)}},
			If:   nil,
			Then: data.Tools,
		},
	}).Then(errors.New("test"))

	_ = result.Json(w, http.StatusOK)
}
