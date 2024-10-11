package middleware

import (
	"net/http"

	"github.com/txix-open/isp-kit/log"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"
)

type HttpError interface {
	WriteError(w http.ResponseWriter) error
}

func ErrorHandler(logger log.Logger) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			err := next.Handle(ctx)
			if err == nil {
				return nil
			}

			logger.Error(ctx.Context(), err)

			httpErr, ok := err.(HttpError)
			if ok {
				return httpErr.WriteError(ctx.ResponseWriter())
			}

			return httperrors.
				New(http.StatusInternalServerError, "internal service error", err).
				WriteError(ctx.ResponseWriter())
		})
	}
}
