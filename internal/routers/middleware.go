package routers

import (
	"context"
	"errors"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/bjwt"
	"github.com/retail-ai-inc/beanq/v4/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"github.com/retail-ai-inc/beanq/v4/helper/ui"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

const authCookieName = "beanq_session"

func Recover() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					_ = debug.Stack()
					writeAPIError(w, http.StatusInternalServerError, berror.InternalServerErrorCode, "internal server error")
				}
			}()
			next(w, r)
		}
	}
}

func HeaderRule() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;")
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
			if err := clearSSEWriteDeadline(w); err != nil && !errors.Is(err, http.ErrNotSupported) {
				writeSSEAuthError(w, flusher, result, name, berror.InternalServerErrorCode, err)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), EventName{}, name))
			next(w, r)
		}
	}
}

func clearSSEWriteDeadline(w http.ResponseWriter) error {
	return http.NewResponseController(w).SetWriteDeadline(time.Time{})
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
				_ = result.Json(w, http.StatusForbidden)
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
	if cookie, err := r.Cookie(authCookieName); err == nil {
		if token := strings.TrimSpace(cookie.Value); token != "" {
			return token, nil
		}
	}
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
	return "", errors.New("missing authorization token")
}

func authorizeRole(r *http.Request, mgo *bmongo.BMongo, ui ui.Ui, username string) error {
	if username == ui.Root.UserName {
		return nil
	}
	roleID := permissionForRequest(r)
	if roleID == 0 {
		return nil
	}
	if mgo == nil {
		return errors.New("mongo is not configured")
	}
	return mgo.CheckRole(r.Context(), username, roleID)
}

var routePermissions = map[string]int{
	"GET /api/v1/dashboard": 1, "GET /api/v1/dashboard/stream": 1,
	"GET /api/v1/schedules": 2, "GET /api/v1/queues": 3, "GET /api/v1/queues/{id}/stream": 3,
	"GET /api/v1/events": 5, "GET /api/v1/events/{id}": 5, "PATCH /api/v1/events/{id}": 6,
	"DELETE /api/v1/events/{id}": 7, "POST /api/v1/events/{id}/retry": 8,
	"GET /api/v1/dead-letters": 9, "DELETE /api/v1/dead-letters/{id}": 11, "POST /api/v1/dead-letters/{id}/retry": 12,
	"GET /api/v1/workflows": 13, "DELETE /api/v1/workflows/{id}": 15,
	"GET /api/v1/redis/info/stream": 18, "GET /api/v1/redis/monitor/stream": 19,
	"GET /api/v1/operation-logs": 21, "DELETE /api/v1/operation-logs/{id}": 30,
	"GET /api/v1/workflow-logs": 13,
	"GET /api/v1/users":         22, "POST /api/v1/users": 23, "DELETE /api/v1/users/{id}": 24, "PATCH /api/v1/users/{id}": 25,
	"GET /api/v1/roles": 26, "POST /api/v1/roles": 27, "DELETE /api/v1/roles/{id}": 28, "PATCH /api/v1/roles/{id}": 29,
	"GET /api/v1/config": 31, "PUT /api/v1/config": 31,
	"GET /api/v1/tenants": 34, "GET /api/v1/tenants/{id}": 34, "POST /api/v1/tenants": 36,
	"PATCH /api/v1/tenants/{id}": 37, "DELETE /api/v1/tenants/{id}": 38,
	"GET /api/v1/mongo": 35,
}

func permissionForRequest(r *http.Request) int {
	pattern := r.Pattern
	if pattern == "" {
		pattern = r.Method + " " + r.URL.Path
	}
	return routePermissions[pattern]
}

func auditOperation(r *http.Request, mgo *bmongo.BMongo, username string) error {
	if mgo == nil {
		return nil
	}
	return mgo.AddOptLog(r.Context(), map[string]any{
		"logType":  bstatus.Operation,
		"expireAt": time.Now(),
		"user":     username,
		"method":   r.Method,
		"uri":      r.URL.Path,
		"addTime":  time.Now(),
		"data":     nil,
	})
}

func requireMongo(mgo *bmongo.BMongo) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if mgo == nil {
				writeAPIError(w, http.StatusServiceUnavailable, berror.InternalServerErrorCode, "mongo is not configured")
				return
			}
			next(w, r)
		}
	}
}

func setAuthCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{Name: authCookieName, Value: token, Path: "/", Expires: expires, MaxAge: int(time.Until(expires).Seconds()), HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode})
}

func clearAuthCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: authCookieName, Path: "/", MaxAge: -1, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode})
}

func writeSSEAuthError(w http.ResponseWriter, flusher http.Flusher, result *response.Result, name, code string, err error) {
	result.Code = code
	result.Msg = err.Error()
	_ = result.EventMsg(w, name)
	flusher.Flush()
}
