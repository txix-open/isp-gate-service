package middleware

import (
	"bufio"
	"bytes"
	"github.com/txix-open/isp-kit/json"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/http/endpoint/buffer"
	"github.com/txix-open/isp-kit/log"
	"isp-gate-service/request"
)

type scSource interface {
	StatusCode() int
}

type writerWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *writerWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	upstream, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("writerWrapper: upstream writer doesn't implement Hijack")
	}
	return upstream.Hijack()
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

func Logger( // nolint:gocognit
	logger log.Logger,
	enableRequestLogging bool,
	enableBodyLogging bool,
	skipBodyLoggingEndpointPrefixes []string,
	enableForceUnescapingUnicode bool,
) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *request.Context) error {
			if !enableRequestLogging {
				return next.Handle(ctx)
			}

			r := ctx.Request()

			logBodyFromCurrenRequest := enableBodyLogging
			if logBodyFromCurrenRequest {
				for _, prefix := range skipBodyLoggingEndpointPrefixes {
					if strings.HasPrefix(ctx.Endpoint(), prefix) {
						logBodyFromCurrenRequest = false
						break
					}
				}
			}

			var scSrc scSource
			var buf *buffer.Buffer
			if logBodyFromCurrenRequest {
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
				r.Body = io.NopCloser(bytes.NewBuffer(buf.RequestBody()))

				scSrc = buf
				ctx.SetResponseWriter(buf)
			} else {
				writer := &writerWrapper{ResponseWriter: ctx.ResponseWriter()}
				scSrc = writer
				ctx.SetResponseWriter(writer)
			}

			originalPath := r.URL.Path //
			// can be changed in http proxy
			err := next.Handle(ctx)

			authData, _ := ctx.GetAuthData()
			fields := []log.Field{
				log.String("httpMethod", r.Method),
				log.String("remoteAddr", r.RemoteAddr),
				log.String("xForwardedFor", r.Header.Get("X-Forwarded-For")),
				log.Int("statusCode", scSrc.StatusCode()),
				log.String("path", originalPath),
				log.String("endpoint", ctx.Endpoint()),
				log.Int("applicationId", authData.ApplicationId),
				log.Int("adminId", ctx.AdminId()),
			}

			if logBodyFromCurrenRequest {
				if enableForceUnescapingUnicode && bytes.Contains(buf.RequestBody(), []byte("\\u")) {
					fields = append(fields, log.ByteString("request", forceUnescapingUnicode(buf.RequestBody())))
				} else {
					fields = append(fields, log.ByteString("request", buf.RequestBody()))
				}

				fields = append(fields, log.ByteString("response", buf.ResponseBody()))
			}
			logger.Debug(ctx.Context(), "log request", fields...)

			return err
		})
	}
}

func forceUnescapingUnicode(data []byte) []byte {
	var body any
	err := json.Unmarshal(data, &body)
	if err != nil {
		return data
	}

	newData, err := json.Marshal(body)
	if err != nil {
		return data
	}

	return newData
}
