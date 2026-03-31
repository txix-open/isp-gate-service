package middleware

import (
	"net/http"

	"isp-gate-service/domain"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"

	"github.com/pkg/errors"
)

type UserAuthenticator interface {
	Authenticate(ctx *request.Context) (*domain.AuthenticateUserResponse, error)
}

func UserAuthenticate(authenticator UserAuthenticator) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			resp, err := authenticator.Authenticate(ctx)
			if err != nil {
				return errors.WithMessage(err, "authenticate user: authenticator error")
			}
			if resp.ShouldSkip {
				return next.Handle(ctx)
			}

			if !resp.Authenticated {
				return httperrors.New(
					http.StatusUnauthorized,
					"invalid user token",
					errors.Errorf("authenticate: %s", resp.ErrorReason),
				)
			}

			ctx.AuthenticateUser(*resp.AuthData)
			return next.Handle(ctx)
		})
	}
}
