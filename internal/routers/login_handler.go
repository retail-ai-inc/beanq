package routers

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bjwt"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/googleAuth"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"

	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Login struct {
	client             redis.UniversalClient
	mgo                *bmongo.BMongo
	prefix             string
	username, password string
	issuer, subject    string
	expiresAt          time.Duration
}

func NewLogin(client redis.UniversalClient, mgo *bmongo.BMongo, prefix string, username, password string, issuer, subject string, expiresAt time.Duration) *Login {
	return &Login{client: client, mgo: mgo, prefix: prefix, username: username, password: password, issuer: issuer, subject: subject, expiresAt: expiresAt}
}

func (t *Login) Login(ctx *bwebframework.BeanContext) error {

	r := ctx.Request
	w := ctx.Writer

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	result, cancel := response.Get()
	defer cancel()

	if username != t.username || password != t.password {
		user, err := t.mgo.CheckUser(r.Context(), username, password)
		if err != nil || user == nil {
			result.Code = berror.AuthExpireCode
			result.Msg = "No permission"
			return result.Json(w, http.StatusUnauthorized)
		}
	}

	claim := bjwt.Claim{
		UserName: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    t.issuer,
			Subject:   t.issuer,
			Audience:  nil,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.expiresAt)),
			NotBefore: nil,
			IssuedAt:  nil,
			ID:        "",
		},
	}

	token, err := bjwt.MakeHsToken(claim)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusInternalServerError)
	}

	client := tool.ClientFac(t.client, t.prefix, "")
	nodeId := client.NodeId(r.Context())

	result.Data = map[string]any{"token": token, "nodeId": nodeId}

	return result.Json(w, http.StatusOK)

}

func (t *Login) GoogleLogin(ctx *bwebframework.BeanContext) error {
	w := ctx.Writer

	gAuth := googleAuth.New()

	state := time.Now().String()
	url := gAuth.AuthCodeUrl(state)
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
	return nil
}

func (t *Login) GoogleCallBack(ctx *bwebframework.BeanContext) error {

	r := ctx.Request
	w := ctx.Writer

	res, cancel := response.Get()
	defer cancel()

	state := r.FormValue("state")
	if state != "test_self" {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return nil
	}

	code := r.FormValue("code")
	auth := googleAuth.New()

	token, err := auth.Exchange(r.Context(), code)

	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusOK)
	}

	userInfo, err := auth.Response(token.AccessToken)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusOK)
	}

	user, err := t.mgo.CheckGoogleUser(r.Context(), userInfo.Email)

	if err != nil || user == nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusOK)
	}

	claim := bjwt.Claim{
		UserName: userInfo.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    t.issuer,
			Subject:   t.subject,
			Audience:  nil,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.expiresAt)),
			NotBefore: nil,
			IssuedAt:  nil,
			ID:        "",
		},
	}
	jwtToken, err := bjwt.MakeHsToken(claim)
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		return res.Json(w, http.StatusOK)
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
	return nil
}
