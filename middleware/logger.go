package middleware

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/integration-system/isp-kit/http/endpoint/buffer"
	"github.com/integration-system/isp-kit/log"
	"github.com/pkg/errors"
	"isp-gate-service/request"
)

type scSource interface {
	StatusCode() int
}

type writerWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *writerWrapper) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func (w *writerWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func Logger(logger log.Logger, enableRequestLogging bool, enableBodyLogging bool) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			if !enableRequestLogging {
				return next.Handle(ctx)
			}

			r := ctx.Request()
			var scSrc scSource
			var buf *buffer.Buffer
			if enableBodyLogging {
				buf = buffer.Acquire(ctx.ResponseWriter())
				defer buffer.Release(buf)

				err := buf.ReadRequestBody(r.Body)
				if err != nil {
					return errors.WithMessage(err, "logger: read request body for logging")
				}
				err = r.Body.Close()
				if err != nil {
					return errors.WithMessage(err, "logger: close request reader")
				}
				r.Body = ioutil.NopCloser(bytes.NewBuffer(buf.RequestBody()))

				scSrc = buf
				ctx.SetResponseWriter(buf)
			} else {
				writer := &writerWrapper{ResponseWriter: ctx.ResponseWriter()}
				scSrc = writer
				ctx.SetResponseWriter(writer)
			}

			err := next.Handle(ctx)

			authData, _ := ctx.GetAuthData()
			fields := []log.Field{
				log.String("http_method", r.Method),
				log.String("remote_addr", r.RemoteAddr),
				log.String("x_forwarded_for", r.Header.Get("X-Forwarded-For")),
				log.Int("status_code", scSrc.StatusCode()),
				log.String("path", r.URL.Path),
				log.String("endpoint", ctx.Endpoint()),
				log.Int("application_id", authData.ApplicationId),
				log.Int("admin_id", ctx.AdminId()),
			}
			if enableBodyLogging {
				fields = append(fields,
					log.ByteString("request", buf.RequestBody()),
					log.ByteString("response", buf.ResponseBody()),
				)
			}
			logger.Debug(ctx.Context(), "log request", fields...)

			return err
		})
	}
}
