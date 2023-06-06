package middleware

import (
	"context"
	"net/http"
	"strings"

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
			token := strings.TrimSpace(ctx.Param(applicationTokenHeader))
			if token == "" {
				return httperrors.New(
					http.StatusUnauthorized,
					"application token required",
					errors.New("authenticate: application token required"),
				)
			}

			resp, err := authenticator.Authenticate(ctx.Context(), token)
			if err != nil {
				return errors.WithMessagef(err, "authenticate: authenticate")
			}
			if !resp.Authenticated {
				return httperrors.New(
					http.StatusUnauthorized,
					"invalid application token",
					errors.WithMessage(errors.New(resp.ErrorReason), "authenticate: authenticate"),
				)
			}
			ctx.Authenticate(request.AuthData(*resp.AuthData))

			return next.Handle(ctx)
		})
	}
}
