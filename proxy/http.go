package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"isp-gate-service/httperrors"
	"isp-gate-service/request"

	"github.com/integration-system/isp-kit/grpc"
	"github.com/integration-system/isp-kit/requestid"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type HttpHostManager interface {
	Next() (string, error)
}

type Http struct {
	hostManager HttpHostManager
	skipAuth    bool
	timeout     time.Duration
}

func NewHttp(hostManager HttpHostManager, skipAuth bool, timeout time.Duration) Http {
	return Http{
		hostManager: hostManager,
		skipAuth:    skipAuth,
		timeout:     timeout,
	}
}

func (p Http) Handle(ctx *request.Context) error {
	host, err := p.hostManager.Next()
	if err != nil {
		return errors.WithMessage(err, "http: next host")
	}

	rawUrl := fmt.Sprintf("http://%s", host) // secure HTTP links are reset connection
	target, err := url.Parse(rawUrl)
	if err != nil {
		return errors.WithMessage(err, "http: parse url")
	}

	request := ctx.Request()
	request.URL.Path = ctx.Endpoint()
	request.Header.Set(grpc.RequestIdHeader, requestid.FromContext(ctx.Context()))
	if !p.skipAuth {
		authData, err := ctx.GetAuthData()
		if err != nil {
			return errors.WithMessage(err, "http: get auth data")
		}
		request.Header.Set(grpc.SystemIdHeader, strconv.Itoa(authData.SystemId))
		request.Header.Set(grpc.DomainIdHeader, strconv.Itoa(authData.DomainId))
		request.Header.Set(grpc.ServiceIdHeader, strconv.Itoa(authData.ServiceId))
		request.Header.Set(grpc.ApplicationIdHeader, strconv.Itoa(authData.ApplicationId))
		if ctx.IsAdminAuthenticated() {
			request.Header.Set(xAdminIdHeader, strconv.Itoa(ctx.AdminId()))
		} else {
			request.Header.Del(xAdminIdHeader)
		}
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	var resultError error
	reverseProxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		resultError = httperrors.New(
			http.StatusServiceUnavailable,
			"upstream is not available",
			errors.WithMessagef(err, "http proxy to %s", host),
		)
	}

	context, cancel := context.WithTimeout(request.Context(), p.timeout)
	defer cancel()
	request = request.WithContext(context)
	reverseProxy.ServeHTTP(ctx.ResponseWriter(), request)

	return resultError
}
