package middleware

import (
	"isp-gate-service/domain"
	"isp-gate-service/request"
	"net/http"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/log"
)

type EntryPointConfig struct {
	PathPrefix             string
	WithPrefix             bool
	ErrorOnUnknownEndpoint bool
	WithLendingSlash       bool
}

type EndpointResolver interface {
	ResolveEndpoint(method string, path string, cfg EntryPointConfig) (*domain.EndpointMeta, error)
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

		endpoint, err := entryPointResolver.ResolveEndpoint(req.Method, req.URL.Path, cfg)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusNotImplemented)
			return
		}

		ctx := request.NewContext(req, writer, endpoint)

		err = next.Handle(ctx)
		if err != nil {
			logger.Error(req.Context(), errors.WithMessage(err, "uncaught error"))
		}
	})
}
