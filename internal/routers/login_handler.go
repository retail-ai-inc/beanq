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
	client redis.UniversalClient
	mgo    *bmongo.BMongo
	prefix string
	ui     Ui
}

func NewLogin(client redis.UniversalClient, mgo *bmongo.BMongo, prefix string, ui Ui) *Login {
	return &Login{client: client, mgo: mgo, prefix: prefix, ui: ui}
}

func (t *Login) Login(ctx *bwebframework.BeanContext) error {

	r := ctx.Request
	w := ctx.Writer

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

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
			Roles:    nil,
		}
		err error
	)

	if username != t.ui.Root.UserName || password != t.ui.Root.Password {
		user, err = t.mgo.CheckUser(r.Context(), username, password)
		if err != nil || user == nil {
			result.Code = berror.AuthExpireCode
			result.Msg = "No permission"
			return result.Json(w, http.StatusUnauthorized)
		}
	}

	claim := bjwt.Claim{
		UserName: username,
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

	token, err := bjwt.MakeHsToken(claim, []byte(t.ui.JwtKey))
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		return result.Json(w, http.StatusInternalServerError)
	}

	client := tool.ClientFac(t.client, t.prefix, "")
	nodeId := client.NodeId(r.Context())

	result.Data = map[string]any{"token": token, "roles": user.Roles, "nodeId": nodeId}

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
