package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/integration-system/isp-kit/log"
	"github.com/pkg/errors"
	"isp-gate-service/request"
)

func Entrypoint(maxReqBodySize int64, next Handler, logger log.Logger, pathPrefix *string) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(writer, req.Body, maxReqBodySize)

		endpoint := req.URL.Path
		if pathPrefix != nil {
			prefix := fmt.Sprintf("%s/", *pathPrefix)
			endpoint = strings.TrimPrefix(req.URL.Path, prefix)
		}
		ctx := request.NewContext(req, writer, endpoint)

		err := next.Handle(ctx)
		if err != nil {
			logger.Error(req.Context(), errors.WithMessage(err, "uncaught error"))
		}
	})
}
