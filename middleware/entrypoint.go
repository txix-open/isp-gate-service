package middleware

import (
	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/log"
	"isp-gate-service/request"
	"net/http"
)

type EntryPointConfig struct {
	PathPrefix string
	WithPrefix bool
}

type EndpointResolver interface {
	ResolveEndpoint(path string, cfg EntryPointConfig) string
}

func Entrypoint(
	maxReqBodySize int64,
	next Handler,
	cfg EntryPointConfig,
	entryPointResolver EndpointResolver,
	logger log.Logger,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(writer, req.Body, maxReqBodySize)

		endpoint := entryPointResolver.ResolveEndpoint(req.URL.Path, cfg)
		ctx := request.NewContext(req, writer, endpoint)

		err := next.Handle(ctx)
		if err != nil {
			logger.Error(req.Context(), errors.WithMessage(err, "uncaught error"))
		}
	})
}
