package middleware

import (
	"context"
	"net/http"

	"github.com/integration-system/isp-kit/log"
	"github.com/pkg/errors"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"
)

type Authorizer interface {
	Authorize(ctx context.Context, applicationId int, endpoint string) (bool, error)
}

func Authorize(authorizer Authorizer, logger log.Logger) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			authData, err := ctx.GetAuthData()
			if err != nil {
				return errors.WithMessage(err, "authorize: get auth data")
			}

			ok, err := authorizer.Authorize(ctx.Context(), authData.ApplicationId, ctx.Endpoint())
			if err != nil {
				return errors.WithMessage(err, "authorize: authorize")
			}
			if !ok && ctx.IsAdminAuthenticated() {
				logger.Info(
					ctx.Context(),
					"bypassing unauthorized request because it has admin privilege",
					log.Int("applicationId", authData.ApplicationId),
					log.Int("adminId", ctx.AdminId()),
				)
				return next.Handle(ctx)
			}
			if !ok {
				return httperrors.New(
					http.StatusForbidden,
					"endpoint is not allowed",
					errors.Errorf("authorize: endpoint '%s' is not allowed for application '%d'", ctx.Endpoint(), authData.ApplicationId),
				)
			}

			return next.Handle(ctx)
		})
	}
}
