package middleware

import (
	"context"
	"net/http"
	"strings"

	"isp-gate-service/domain"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"

	"github.com/pkg/errors"
)

const (
	adminAuthHeader = "x-auth-admin"
)

type AdminAuthenticator interface {
	AdminAuthenticate(ctx context.Context, token string) (*domain.AdminAuthenticateResponse, error)
}

func AdminAuthenticate(auth AdminAuthenticator) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			adminToken := strings.TrimSpace(ctx.Request().Header.Get(adminAuthHeader))
			if adminToken == "" {
				return next.Handle(ctx)
			}
			resp, err := auth.AdminAuthenticate(ctx.Context(), adminToken)
			if err != nil {
				return httperrors.New(
					http.StatusUnauthorized,
					"invalid admin token",
					errors.WithMessage(err, "admin authenticate: authenticate"),
				)
			}
			if !resp.Authenticated {
				return httperrors.New(
					http.StatusUnauthorized,
					"invalid admin token",
					errors.WithMessage(errors.New(resp.ErrorReason), "admin authenticate: authenticate"),
				)
			}
			ctx.AuthenticateAdmin(resp.AdminId, adminToken)
			return next.Handle(ctx)
		})
	}
}
