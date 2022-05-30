package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/integration-system/isp-kit/grpc"
	"github.com/pkg/errors"
	"isp-gate-service/domain"
	"isp-gate-service/middleware"
)

type HttpHostManager interface {
	Next() (string, error)
}

type Http struct {
	hostManager HttpHostManager
}

func NewHttp(hostManager HttpHostManager) Http {
	return Http{
		hostManager: hostManager,
	}
}

func (p Http) Handle(ctx *middleware.Context) error {
	host, err := p.hostManager.Next()
	if err != nil {
		return errors.WithMessage(err, "next host")
	}

	rawUrl := fmt.Sprintf("http://%s", host) // secure HTTP links are reset connection
	target, err := url.Parse(rawUrl)
	if err != nil {
		return errors.WithMessage(err, "parse url")
	}

	ctx.Request.URL.Path = ctx.Path
	ctx.Request.Header.Set(grpc.ApplicationIdHeader, strconv.Itoa(ctx.AppId))
	ctx.Request.Header.Set(adminIdHeader, strconv.Itoa(ctx.AdminId))
	reverseProxy := httputil.NewSingleHostReverseProxy(target)

	var resultError error
	reverseProxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		resultError = errors.WithMessage(err, "serve http")

		ctx.ResponseWriter.WriteHeader(http.StatusServiceUnavailable)
		_, err = ctx.ResponseWriter.Write([]byte(domain.ServiceIsNotAvailableErrorMessage))
		if err != nil {
			resultError = errors.WithMessagef(resultError, "write error: %v", err)
		}
	}
	reverseProxy.ServeHTTP(ctx.ResponseWriter, ctx.Request)
	return resultError
}
