package middleware

import (
	"isp-gate-service/request"
)

func JsonResponseUtf8Charset(applicationIds map[int]bool) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			authData, _ := ctx.GetAuthData()
			ctx.EnableHttpJsonUtf8Charset(applicationIds[authData.ApplicationId])
			return next.Handle(ctx)
		})
	}
}
