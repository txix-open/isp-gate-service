package middleware

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"
)

type AdminMethodStore interface {
	IsInnerEndpoint(endpoint string) bool
	RequiredAdminPermission(endpoint string) (string, bool)
}

type AdminAuthorizer interface {
	AdminAuthorize(ctx context.Context, adminId int, permission string) (bool, error)
}

func AdminAuthorize(store AdminMethodStore, authorizer AdminAuthorizer) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			isInner := store.IsInnerEndpoint(ctx.Endpoint())
			if !isInner {
				return next.Handle(ctx)
			}

			if !ctx.IsAdminAuthenticated() {
				return httperrors.New(
					http.StatusForbidden,
					"admin authentication required",
					errors.Errorf("admin authorization: admin authentication required for '%s'", ctx.Endpoint()),
				)
			}

			requiredPerm, ok := store.RequiredAdminPermission(ctx.Endpoint())
			if !ok {
				return next.Handle(ctx)
			}

			ok, err := authorizer.AdminAuthorize(ctx.Context(), ctx.AdminId(), requiredPerm)
			if err != nil {
				return errors.WithMessage(err, "admin authorization: authorize")
			}
			if !ok {
				return httperrors.New(
					http.StatusForbidden,
					"endpoint is not allowed",
					errors.Errorf(
						"admin authorization: endpoint '%s' requires '%s' permission, but admin '%d' doesn't have it",
						ctx.Endpoint(), requiredPerm, ctx.AdminId(),
					),
				)
			}

			return next.Handle(ctx)
		})
	}
}
