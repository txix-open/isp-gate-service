package middleware

import (
	"context"
	"net/http"

	"isp-gate-service/domain"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"

	"github.com/pkg/errors"
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
			token, appName, err := extractToken(ctx)
			if err != nil {
				return err
			}
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
					errors.New("authenticate: application name mismatch"),
				)
			}

			ctx.Authenticate(request.AuthData(*resp.AuthData))
			return next.Handle(ctx)
		})
	}
}

func extractToken(ctx *request.Context) (string, string, error) {
	var (
		appName string
		token   string
		ok      bool
	)

	token = ctx.Param(applicationTokenHeader)
	if token != "" {
		return token, "", nil
	}

	appName, token, ok = ctx.Request().BasicAuth()
	if ok && appName == "" {
		return "", "", httperrors.New(
			http.StatusUnauthorized,
			"application name required",
			errors.New("authenticate: application name required on basic auth"),
		)
	}

	return token, appName, nil
}
