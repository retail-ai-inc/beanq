package routers

import (
	"github.com/go-redis/redis/v8"
	"github.com/retail-ai-inc/beanq/v3/helper/berror"
	"github.com/retail-ai-inc/beanq/v3/helper/bjwt"
	"github.com/retail-ai-inc/beanq/v3/helper/bmongo"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/bwebframework"
	"github.com/retail-ai-inc/beanq/v3/helper/response"
	"github.com/retail-ai-inc/beanq/v3/helper/ui"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/cast"
	"net/http"
	"strings"
	"time"
)

func HeaderRule(next bwebframework.HandleFunc) bwebframework.HandleFunc {
	return func(ctx *bwebframework.BeanContext) error {
		ctx.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		ctx.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline';")
		ctx.Writer.Header().Set("X-Frame-Options", "SAMEORIGIN")
		ctx.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		return next(ctx)
	}
}

func MigrateMiddleWare(next bwebframework.HandleFunc, client redis.UniversalClient, x *bmongo.BMongo, prefix string, ui ui.Ui) bwebframework.HandleFunc {
	return HeaderRule(Auth(next, client, x, prefix, ui))
}

func Auth(next bwebframework.HandleFunc, client redis.UniversalClient, x *bmongo.BMongo, prefix string, ui ui.Ui) bwebframework.HandleFunc {
	return func(ctx *bwebframework.BeanContext) error {

		result, cancelr := response.Get()
		defer cancelr()
		request := ctx.Request
		writer := ctx.Writer

		var (
			err   error
			token *bjwt.Claim
		)

		auth := request.Header.Get("Beanq-Authorization")
		if auth != "" {
			strs := strings.Split(auth, " ")
			if len(strs) < 2 {
				result.Code = berror.InternalServerErrorCode
				result.Msg = "missing parameter"
				return result.Json(writer, http.StatusInternalServerError)
			}
			auth = strs[1]
		} else {
			auth = request.FormValue("token")
		}
		token, err = bjwt.ParseHsToken(auth, []byte(ui.JwtKey))
		if err != nil {
			result.Code = berror.InternalServerErrorMsg
			result.Msg = err.Error()
			return result.Json(writer, http.StatusUnauthorized)
		}
		// Check that the username must be an email address
		if token.UserName != ui.Root.UserName {
			if _, err := mail.ParseEmail(token.UserName); err != nil {
				result.Code = berror.MissParameterCode
				result.Msg = err.Error()
				return result.Json(writer, http.StatusInternalServerError)
			}
		}

		if token.UserName != ui.Root.UserName {
			roleId := cast.ToInt(request.Header.Get("X-Role-Id"))
			if err := x.CheckRole(request.Context(), token.UserName, roleId); err != nil {
				result.Code = berror.AuthExpireCode
				result.Msg = err.Error()
				return result.Json(writer, http.StatusUnauthorized)
			}
		}

		if err := x.AddOptLog(request.Context(), map[string]any{"logType": bstatus.Operation, "user": token.UserName, "uri": request.RequestURI, "addTime": time.Now(), "data": nil}); err != nil {
			result.Code = berror.InternalServerErrorCode
			result.Msg = err.Error()
			return result.Json(writer, http.StatusInternalServerError)
		}
		return next(ctx)
	}
}
