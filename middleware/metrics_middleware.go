package middleware

import (
	"isp-gate-service/request"
	"time"

	"github.com/txix-open/isp-kit/metrics/http_metrics"
)

func Metrics(storage *http_metrics.ServerStorage) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			pathSchema := ctx.EndpointMeta().PathSchema
			if pathSchema == "" {
				return next.Handle(ctx)
			}

			r := ctx.Request()

			start := time.Now()
			err := next.Handle(ctx)
			if err != nil {
				return err
			}

			storage.ObserveDuration(r.Method, pathSchema, time.Since(start))

			return nil
		})
	}
}
