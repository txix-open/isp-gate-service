package middleware

import (
	"net/http"

	"isp-gate-service/domain"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/log"
)

type UserAuthenticator interface {
	Authenticate(ctx *request.Context) (*domain.AuthenticateUserResponse, error)
}

func UserAuthenticate(authenticator UserAuthenticator, logger log.Logger) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			resp, err := authenticator.Authenticate(ctx)
			switch {
			case errors.Is(err, domain.ErrEmptyUserToken):
				return httperrors.New(
					http.StatusUnauthorized,
					"user token required",
					errors.New("authenticate: user token required"),
				)
			case err != nil:
				return errors.WithMessage(err, "authenticate user: authenticator error")
			}

			if resp.SkipUserAuth {
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

			if ctx.SkipAppAuth() {
				logger.Debug(
					ctx.Context(),
					"skip app auth is set. All steps that require app auth data are skipped",
				)
			}
			return next.Handle(ctx)
		})
	}
}
