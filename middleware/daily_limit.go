package middleware

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"
)

type DailyLimitChecker interface {
	IncrementAndCheck(ctx context.Context, applicationId int) (bool, error)
}

func DailyLimit(checker DailyLimitChecker) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			authData, err := ctx.GetAuthData()
			if err != nil {
				return errors.WithMessage(err, "daily limit: get auth data")
			}

			ok, err := checker.IncrementAndCheck(ctx.Context(), authData.ApplicationId)
			if err != nil {
				return errors.WithMessage(err, "daily limit: increment and check")
			}
			if !ok {
				return httperrors.New(
					http.StatusTooManyRequests,
					"daily requests limit has been reached",
					errors.Errorf("daily limit: daily requests limit has been reached for application '%d'", authData.ApplicationId),
				)
			}

			return next.Handle(ctx)
		})
	}
}
