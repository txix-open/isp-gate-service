package middleware

import (
	"strings"

	"github.com/integration-system/isp-kit/log"
	"github.com/integration-system/isp-kit/requestid"
	"isp-gate-service/request"
)

const (
	requestIdHeader = "x-request-id"
)

func RequestId() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			requestId := strings.TrimSpace(ctx.Request().Header.Get(requestIdHeader))
			if requestId == "" {
				requestId = requestid.Next()
			}

			context := requestid.ToContext(ctx.Context(), requestId)
			context = log.ToContext(context, log.String("requestId", requestId))

			ctx.SetContext(context)

			return next.Handle(ctx)
		})
	}
}
