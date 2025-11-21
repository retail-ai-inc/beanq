package routers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bjwt"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"github.com/retail-ai-inc/beanq/v4/helper/ui"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/cast"
)

func Recover() {
	// todo
}

func HeaderRule(next func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline';")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next(w, r)
	}
}

func MigrateMiddleWare(next func(w http.ResponseWriter, r *http.Request), client redis.UniversalClient, x *bmongo.BMongo, prefix string, ui ui.Ui) func(w http.ResponseWriter, r *http.Request) {
	return HeaderRule(Auth(next, client, x, prefix, ui))
}

func MigrateSSE(next func(w http.ResponseWriter, r *http.Request), client redis.UniversalClient, x *bmongo.BMongo, prefix string, ui ui.Ui, name string) func(w http.ResponseWriter, r *http.Request) {
	return HeaderRule(AuthSSE(next, client, x, prefix, ui, name))
}

func AuthSSE(next func(w http.ResponseWriter, r *http.Request), client redis.UniversalClient, x *bmongo.BMongo, prefix string, ui ui.Ui, name string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		result, cancelr := response.Get()
		defer cancelr()

		var (
			err   error
			token *bjwt.Claim
		)

		auth := r.FormValue("token")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "server error", http.StatusInternalServerError)
			flusher.Flush()
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		token, err = bjwt.ParseHsToken(auth, []byte(ui.JwtKey))
		if err != nil {
			result.Code = berror.AuthExpireCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, name)
			flusher.Flush()
			return
		}
		// Check that the username must be an email address
		if token.UserName != ui.Root.UserName {
			if _, err := mail.ParseEmail(token.UserName); err != nil {
				result.Code = berror.MissParameterCode
				result.Msg = err.Error()
				_ = result.EventMsg(w, name)
				flusher.Flush()
				return
			}
		}

		if token.UserName != ui.Root.UserName {
			roleId := cast.ToInt(r.Header.Get("X-Role-Id"))
			if roleId > 0 {
				if err := x.CheckRole(r.Context(), token.UserName, roleId); err != nil {
					result.Code = berror.AuthExpireCode
					result.Msg = err.Error()
					_ = result.EventMsg(w, name)
					flusher.Flush()
					return
				}
			}
		}

		if err := x.AddOptLog(r.Context(), map[string]any{"logType": bstatus.Operation, "expireAt": time.Now(), "user": token.UserName, "uri": r.RequestURI, "addTime": time.Now(), "data": nil}); err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.EventMsg(w, name)
			flusher.Flush()
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), EventName{}, name))

		next(w, r)
	}
}

func Auth(next func(w http.ResponseWriter, r *http.Request), client redis.UniversalClient, x *bmongo.BMongo, prefix string, ui ui.Ui) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		result, cancelr := response.Get()
		defer cancelr()

		var (
			err   error
			token *bjwt.Claim
		)

		auth := r.Header.Get("Beanq-Authorization")
		if auth != "" {
			strs := strings.Split(auth, " ")
			if len(strs) < 2 {
				result.Code = berror.AuthExpireCode
				result.Msg = "missing parameter"
				_ = result.Json(w, http.StatusInternalServerError)
				return
			}
			auth = strs[1]
		} else {
			auth = r.FormValue("token")
		}
		token, err = bjwt.ParseHsToken(auth, []byte(ui.JwtKey))
		if err != nil {
			result.Code = berror.AuthExpireCode
			result.Msg = err.Error()
			_ = result.Json(w, http.StatusUnauthorized)
			return
		}
		// Check that the username must be an email address
		if token.UserName != ui.Root.UserName {
			if _, err := mail.ParseEmail(token.UserName); err != nil {
				result.Code = berror.MissParameterCode
				result.Msg = err.Error()
				_ = result.Json(w, http.StatusInternalServerError)
				return
			}
		}

		if token.UserName != ui.Root.UserName {
			roleId := cast.ToInt(r.Header.Get("X-Role-Id"))
			if roleId > 0 {
				if err := x.CheckRole(r.Context(), token.UserName, roleId); err != nil {
					result.Code = berror.AuthExpireCode
					result.Msg = err.Error()
					_ = result.Json(w, http.StatusUnauthorized)
					return
				}
			}
		}

		if err := x.AddOptLog(r.Context(), map[string]any{"logType": bstatus.Operation, "expireAt": time.Now(), "user": token.UserName, "uri": r.RequestURI, "addTime": time.Now(), "data": nil}); err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			_ = result.Json(w, http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), "username", token.UserName)
		r = r.WithContext(ctx)
		next(w, r)
	}
}
