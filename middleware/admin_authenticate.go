package middleware

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"
)

const (
	adminAuthHeader = "x-auth-admin"
)

type AdminAuthenticator interface {
	Authenticate(token string) (int, error)
}

func AdminAuthenticate(auth AdminAuthenticator) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			adminToken := strings.TrimSpace(ctx.Request().Header.Get(adminAuthHeader))
			if adminToken == "" {
				return next.Handle(ctx)
			}
			adminId, err := auth.Authenticate(adminToken)
			if err != nil {
				return httperrors.New(
					http.StatusUnauthorized,
					"invalid admin token",
					errors.WithMessage(err, "admin authenticate: authenticate"),
				)
			}
			ctx.AuthenticateAdmin(adminId, adminToken)
			return next.Handle(ctx)
		})
	}
}
