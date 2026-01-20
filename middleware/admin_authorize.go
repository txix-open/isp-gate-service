package middleware

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"
)

type AdminAuthorizer interface {
	AdminAuthorize(ctx context.Context, adminId int, permission string) (bool, error)
}

func AdminAuthorize(authorizer AdminAuthorizer) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			endpointMeta := ctx.EndpointMeta()

			isInner := endpointMeta.Inner
			if !isInner {
				return next.Handle(ctx)
			}

			if !ctx.IsAdminAuthenticated() {
				return httperrors.New(
					http.StatusForbidden,
					"admin authentication required",
					errors.Errorf("admin authorization: admin authentication required for '%s'",
						endpointMeta.Endpoint),
				)
			}

			requiredPerm := endpointMeta.RequiredAdminPermission
			if requiredPerm == "" {
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
						endpointMeta.Endpoint, requiredPerm, ctx.AdminId(),
					),
				)
			}

			return next.Handle(ctx)
		})
	}
}
