package middleware

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"isp-gate-service/domain"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"
)

const (
	applicationTokenHeader = "x-application-token"
)

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*domain.AuthenticateResponse, error)
}

func Authenticate(authenticator Authenticator) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			token, appName := extractToken(ctx)
			if token == "" {
				return httperrors.New(
					http.StatusUnauthorized,
					"application token required",
					errors.New("authenticate: application token required"),
				)
			}

			resp, err := authenticator.Authenticate(ctx.Context(), token)
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
					errors.New("authenticate: basic auth failed"),
				)
			}

			ctx.Authenticate(request.AuthData(*resp.AuthData))
			return next.Handle(ctx)
		})
	}
}

func extractToken(ctx *request.Context) (string, string) {
	appName, token, ok := ctx.Request().BasicAuth()
	if ok {
		return token, appName
	}

	return ctx.Param(applicationTokenHeader), ""
}
