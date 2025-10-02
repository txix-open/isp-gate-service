package middleware

import (
	"strings"

	"isp-gate-service/request"

	"github.com/txix-open/isp-kit/log"
	"github.com/txix-open/isp-kit/requestid"
)

const (
	requestIdHeader = "x-request-id"
)

func RequestId(forwardClientRequestId bool, forwardReqIdByApp map[int]bool) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			requestId := requestid.Next()
			clientRequestId := strings.TrimSpace(ctx.Request().Header.Get(requestIdHeader))

			logFields := make([]log.Field, 0)
			if clientRequestId != "" {
				logFields = append(logFields, log.String("clientRequestId", clientRequestId))

				authData, _ := ctx.GetAuthData()
				if forwardClientRequestId || forwardReqIdByApp[authData.ApplicationId] {
					requestId = clientRequestId
				}
			}
			logFields = append(logFields, log.String("requestId", requestId))

			context := requestid.ToContext(ctx.Context(), requestId)
			context = log.ToContext(context, logFields...)

			ctx.SetContext(context)
			return next.Handle(ctx)
		})
	}
}
