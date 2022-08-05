package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"isp-gate-service/domain"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"
)

type Throttler interface {
	AllowRateLimit(ctx context.Context, applicationId int) (*domain.RateLimitResult, error)
}

func Throttling(throttler Throttler) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			authData, err := ctx.GetAuthData()
			if err != nil {
				return errors.WithMessage(err, "throttling: get auth data")
			}

			result, err := throttler.AllowRateLimit(ctx.Context(), authData.ApplicationId)
			if err != nil {
				return errors.WithMessage(err, "throttling: allow rate limit")
			}
			if !result.Allow {
				return httperrors.New(
					http.StatusTooManyRequests,
					fmt.Sprintf("rate limit has been reached, try after %dms", result.RetryAfter.Milliseconds()),
					errors.Errorf("throttling: rate limit has been reached for application '%d'", authData.ApplicationId),
				)
			}

			return next.Handle(ctx)
		})
	}
}
