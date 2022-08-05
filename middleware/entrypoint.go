package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/integration-system/isp-kit/log"
	"github.com/pkg/errors"
	"isp-gate-service/request"
)

func Entrypoint(maxReqBodySize int64, next Handler, pathPrefix string, logger log.Logger) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(writer, req.Body, maxReqBodySize)

		prefix := fmt.Sprintf("%s/", pathPrefix)
		endpoint := strings.TrimPrefix(req.URL.String(), prefix)
		ctx := request.NewContext(req, writer, endpoint)

		err := next.Handle(ctx)
		if err != nil {
			logger.Error(req.Context(), errors.WithMessage(err, "uncaught error"))
		}
	})
}
