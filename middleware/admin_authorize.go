package middleware

import (
	"net/http"

	"github.com/pkg/errors"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"
)

type AdminMethodStore interface {
	IsInnerMethod(path string) bool
}

func AdminAuthorize(store AdminMethodStore) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			isInner := store.IsInnerMethod(ctx.Endpoint())
			if !isInner {
				return next.Handle(ctx)
			}
			if ctx.IsAdminAuthenticated() {
				return next.Handle(ctx)
			}
			return httperrors.New(
				http.StatusForbidden,
				"admin authentication required",
				errors.Errorf("admin authorization: admin authentication required for '%s'", ctx.Endpoint()),
			)
		})
	}
}
