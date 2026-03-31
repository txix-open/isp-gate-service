package middleware

import (
	"context"
	"net/http"

	"isp-gate-service/domain"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"

	"github.com/pkg/errors"
)

type Authenticator interface {
	Authenticate(ctx context.Context, endpoint string, token string) (*domain.AuthenticateResponse, error)
}

type TokenExtractor interface {
	ExtractToken(ctx *request.Context) (string, string, error)
}

func Authenticate(
	tokenExtractor TokenExtractor,
	authenticator Authenticator,
) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			token, appName, err := tokenExtractor.ExtractToken(ctx)
			if err != nil {
				return errors.WithMessage(err, "extract token")
			}
			if token == "" {
				return httperrors.New(
					http.StatusUnauthorized,
					"application token required",
					errors.New("authenticate: application token required"),
				)
			}

			resp, err := authenticator.Authenticate(ctx.Context(), ctx.EndpointMeta().NormalizedEndpoint, token)
			if err != nil {
				return errors.WithMessage(err, "authenticate: authenticator error")
			}

			if !resp.Authenticated {
				return httperrors.New(
					http.StatusUnauthorized,
					"invalid application token",
					errors.Errorf("authenticate: %s", resp.ErrorReason),
				)
			}

			if appName != "" && resp.AuthData.AppName != appName {
				return httperrors.New(
					http.StatusUnauthorized,
					"invalid application token",
					errors.New("authenticate: application name mismatch"),
				)
			}

			ctx.Authenticate(*resp.AuthData)
			return next.Handle(ctx)
		})
	}
}
