package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/log"
	"isp-gate-service/request"
)

type EntryPointConfig struct {
	PathPrefix string
	WithPrefix bool
}

func Entrypoint(maxReqBodySize int64, next Handler, logger log.Logger, cfg EntryPointConfig) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(writer, req.Body, maxReqBodySize)

		endpoint := req.URL.Path
		if !cfg.WithPrefix {
			prefix := fmt.Sprintf("%s/", cfg.PathPrefix)
			endpoint = strings.TrimPrefix(req.URL.Path, prefix)
		}
		ctx := request.NewContext(req, writer, endpoint)

		err := next.Handle(ctx)
		if err != nil {
			logger.Error(req.Context(), errors.WithMessage(err, "uncaught error"))
		}
	})
}
