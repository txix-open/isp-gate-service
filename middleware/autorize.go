package middleware

import (
	"github.com/pkg/errors"
	"isp-gate-service/domain"
)

type AdminAuthorize interface {
	Authorize(id int, path string) error
}

func Authorize(adminAuthorize AdminAuthorize) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) error {
			if !ctx.authenticated {
				return errors.WithMessage(domain.ErrAuthorize, "authentication is required")
			}

			err := adminAuthorize.Authorize(ctx.AdminId, ctx.Path)
			if err != nil {
				return errors.WithMessagef(domain.ErrAuthorize, "authorize admin: %v", err)
			}

			return next.Handle(ctx)
		})
	}
}
