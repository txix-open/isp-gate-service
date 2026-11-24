package middleware

import (
	"net/http"

	"isp-gate-service/httperrors"
	"isp-gate-service/request"

	"github.com/txix-open/isp-kit/errors"
)

type UserAuthorizer interface {
	UserAuthorize(ctx *request.Context) (bool, error)
}

func UserAuthorize(authorizer UserAuthorizer) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			endpointMeta := ctx.EndpointMeta()
			requiredPerm := endpointMeta.RequiredUserPermission
			if requiredPerm == "" {
				return next.Handle(ctx)
			}

			userAuthData, err := ctx.GetUserAuthData()
			if err != nil {
				return httperrors.New(
					http.StatusUnauthorized,
					"user authentication required",
					errors.Errorf("user authorization: user authentication required for '%s'",
						endpointMeta.Endpoint,
					),
				)
			}

			ok, err := authorizer.UserAuthorize(ctx)
			if err != nil {
				return errors.WithMessage(err, "user authorization: authorize")
			}
			if !ok {
				return httperrors.New(
					http.StatusForbidden,
					"endpoint is not allowed",
					errors.Errorf(
						"user authorization: endpoint '%s' requires '%s' permission, but user '%s' doesn't have it",
						endpointMeta.Endpoint, requiredPerm, userAuthData.Identity,
					),
				)
			}

			return next.Handle(ctx)
		})
	}
}
