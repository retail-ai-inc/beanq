package routers

import (
	"net/http"
	"strings"

	"github.com/retail-ai-inc/beanq/client/internal/jwtx"
	"github.com/retail-ai-inc/beanq/client/internal/simple_router"
)

func Auth(next simple_router.HandlerFunc) simple_router.HandlerFunc {

	return func(ctx *simple_router.Context) error {
		result := resultPool.Get().(*Result)
		defer func() {
			result.Reset()
			resultPool.Put(result)
		}()

		req := ctx.Request()

		auth := req.Header.Get("Beanq-Authorization")

		strs := strings.Split(auth, " ")
		if len(strs) < 2 {
			// return data format err
			result.Code = "1001"
			result.Msg = "missing parameter"
			return ctx.Json(http.StatusInternalServerError, result)
		}

		token, err := jwtx.ParseRsaToken(strs[1])
		if err != nil {
			result.Code = "1001"
			result.Msg = err.Error()
			return ctx.Json(http.StatusUnauthorized, result)
		}
		//
		_, err = token.Claims.GetExpirationTime()
		if err != nil {
			result.Msg = err.Error()
			return ctx.Json(http.StatusUnauthorized, result)
		}

		return next(ctx)
	}
}