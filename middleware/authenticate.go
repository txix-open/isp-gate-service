package middleware

import (
	"github.com/pkg/errors"
	"isp-gate-service/domain"
)

const (
	adminTokenHeader = "x-auth-admin"
)

type AdminAuthenticate interface {
	Authenticate(token string) (int, error)
}

func Authenticate(adminAuthenticate AdminAuthenticate) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) error {
			ctx.AppId = 1 // todo application

			adminToken := ctx.Request.Header.Get(adminTokenHeader)
			adminId, err := adminAuthenticate.Authenticate(adminToken)
			if err != nil {
				return errors.WithMessagef(domain.ErrAuthenticate, "admin athenticate: %v", err)
			}
			ctx.AdminId = adminId

			ctx.authenticated = true

			return next.Handle(ctx)
		})
	}
}
