package middleware

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/log"
	"isp-gate-service/request"
)

type EntryPointConfig struct {
	PathPrefix string
	WithPrefix bool
	IsGrpcPath bool
}

func Entrypoint(maxReqBodySize int64, next Handler, logger log.Logger, cfg EntryPointConfig) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(writer, req.Body, maxReqBodySize)
		ctx := request.NewContext(req, writer, getEndpoint(req, cfg))
		err := next.Handle(ctx)
		if err != nil {
			logger.Error(req.Context(), errors.WithMessage(err, "uncaught error"))
		}
	})
}

func getEndpoint(req *http.Request, cfg EntryPointConfig) string {
	if cfg.WithPrefix {
		return req.URL.Path
	}
	if cfg.IsGrpcPath {
		cfg.PathPrefix += "/"
	}
	return strings.TrimPrefix(req.URL.Path, cfg.PathPrefix)
}
