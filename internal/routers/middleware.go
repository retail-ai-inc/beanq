package routers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

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

func HeaderRule() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline';")
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next(w, r)
		}
	}
}

func AuthSSE(x *bmongo.BMongo, ui ui.Ui, name string) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming unsupported", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			result, cancelr := response.Get()
			defer cancelr()

			token, err := authenticate(r, ui)
			if err != nil {
				writeSSEAuthError(w, flusher, result, name, berror.AuthExpireCode, err)
				return
			}
			if err := authorizeRole(r, x, ui, token.UserName); err != nil {
				writeSSEAuthError(w, flusher, result, name, berror.AuthExpireCode, err)
				return
			}
			if err := auditOperation(r, x, token.UserName); err != nil {
				writeSSEAuthError(w, flusher, result, name, berror.InternalServerErrorCode, err)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), EventName{}, name))
			next(w, r)
		}
	}
}

type contextKey struct{}

var UserName = &contextKey{}

func Auth(x *bmongo.BMongo, ui ui.Ui) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			result, cancelr := response.Get()
			defer cancelr()

			token, err := authenticate(r, ui)
			if err != nil {
				result.Code = berror.AuthExpireCode
				result.Msg = err.Error()
				_ = result.Json(w, http.StatusUnauthorized)
				return
			}
			if err := authorizeRole(r, x, ui, token.UserName); err != nil {
				result.Code = berror.AuthExpireCode
				result.Msg = err.Error()
				_ = result.Json(w, http.StatusUnauthorized)
				return
			}
			if err := auditOperation(r, x, token.UserName); err != nil {
				result.Code = berror.InternalServerErrorCode
				result.Msg = err.Error()
				_ = result.Json(w, http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), UserName, token.UserName)
			next(w, r.WithContext(ctx))
		}
	}
}

func authenticate(r *http.Request, ui ui.Ui) (*bjwt.Claim, error) {
	auth, err := authToken(r)
	if err != nil {
		return nil, err
	}
	token, err := bjwt.ParseHsToken(auth, []byte(ui.JwtKey))
	if err != nil {
		return nil, err
	}
	if token.UserName != ui.Root.UserName {
		if _, err := mail.ParseEmail(token.UserName); err != nil {
			return nil, err
		}
	}
	return token, nil
}

func authToken(r *http.Request) (string, error) {
	for _, header := range []string{"Authorization", "Beanq-Authorization"} {
		auth := strings.TrimSpace(r.Header.Get(header))
		if auth == "" {
			continue
		}
		fields := strings.Fields(auth)
		if len(fields) == 1 {
			return fields[0], nil
		}
		if len(fields) == 2 && strings.EqualFold(fields[0], "Bearer") {
			return fields[1], nil
		}
		return "", errors.New("invalid authorization header")
	}
	if token := strings.TrimSpace(r.FormValue("token")); token != "" {
		return token, nil
	}
	return "", errors.New("missing authorization token")
}

func authorizeRole(r *http.Request, mgo *bmongo.BMongo, ui ui.Ui, username string) error {
	if username == ui.Root.UserName {
		return nil
	}
	roleID := cast.ToInt(r.Header.Get("X-Role-Id"))
	if roleID <= 0 {
		return nil
	}
	if mgo == nil {
		return errors.New("mongo is not configured")
	}
	return mgo.CheckRole(r.Context(), username, roleID)
}

func auditOperation(r *http.Request, mgo *bmongo.BMongo, username string) error {
	if mgo == nil {
		return nil
	}
	return mgo.AddOptLog(r.Context(), map[string]any{
		"logType":  bstatus.Operation,
		"expireAt": time.Now(),
		"user":     username,
		"uri":      r.RequestURI,
		"addTime":  time.Now(),
		"data":     nil,
	})
}

func writeSSEAuthError(w http.ResponseWriter, flusher http.Flusher, result *response.Result, name, code string, err error) {
	result.Code = code
	result.Msg = err.Error()
	_ = result.EventMsg(w, name)
	flusher.Flush()
}
