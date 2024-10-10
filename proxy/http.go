package proxy

import (
	"fmt"
	"net"
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

// nolint:gomnd
var (
	httpTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: defaultTransportDialContext(&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          512,
		MaxIdleConnsPerHost:   32,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
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
	err = setHttpHeaders(ctx, request.Header, p.skipAuth)
	if err != nil {
		return err
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	reverseProxy.Transport = httpTransport
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

func setHttpHeaders(ctx *request.Context, header http.Header, skipAuth bool) error {
	header.Set(grpc.RequestIdHeader, requestid.FromContext(ctx.Context()))
	if !skipAuth {
		authData, err := ctx.GetAuthData()
		if err != nil {
			return errors.WithMessage(err, "http: get auth data")
		}
		header.Set(grpc.SystemIdHeader, strconv.Itoa(authData.SystemId))
		header.Set(grpc.DomainIdHeader, strconv.Itoa(authData.DomainId))
		header.Set(grpc.ServiceIdHeader, strconv.Itoa(authData.ServiceId))
		header.Set(grpc.ApplicationIdHeader, strconv.Itoa(authData.ApplicationId))
		if ctx.IsAdminAuthenticated() {
			header.Set(xAdminIdHeader, strconv.Itoa(ctx.AdminId())) //nolint:canonicalheader
		} else {
			header.Del(xAdminIdHeader) //nolint:canonicalheader
		}
	}
	return nil
}

func defaultTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return dialer.DialContext
}
