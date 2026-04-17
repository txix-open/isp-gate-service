package middleware

import (
	"context"
	"strings"

	"isp-gate-service/request"

	"github.com/txix-open/isp-kit/log"
	"github.com/txix-open/isp-kit/requestid"
)

const (
	requestIdHeader = "x-request-id"
)

func RequestId() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			requestId := requestid.Next()

			context := requestid.ToContext(ctx.Context(), requestId)
			context = writeLogField(context, log.String("requestId", requestId))

			ctx.SetContext(context)
			return next.Handle(ctx)
		})
	}
}

func ClientRequestId(forwardClientRequestId bool, forwardReqIdByApp map[int]bool) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			requestId := requestid.FromContext(ctx.Context())
			if requestId == "" {
				requestId = requestid.Next()
			}

			clientRequestId := strings.TrimSpace(ctx.Request().Header.Get(requestIdHeader))

			clientLogFields := make([]log.Field, 0, 1)
			if clientRequestId != "" {
				authData, _ := ctx.GetAuthData()
				if forwardClientRequestId || forwardReqIdByApp[authData.ApplicationId] {
					requestId = clientRequestId
				}

				clientLogFields = append(clientLogFields, log.String("clientRequestId", clientRequestId))
			}

			context := requestid.ToContext(ctx.Context(), requestId)
			context = writeLogField(context, log.String("requestId", requestId))
			context = log.ToContext(context, clientLogFields...)

			ctx.SetContext(context)
			return next.Handle(ctx)
		})
	}
}

func writeLogField(ctx context.Context, field log.Field) context.Context {
	for _, f := range log.ContextLogValues(ctx) {
		if f.Key == field.Key {
			return log.RewriteContextField(ctx, field)
		}
	}

	return log.ToContext(ctx, field)
}
