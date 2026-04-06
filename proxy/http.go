package proxy

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"isp-gate-service/httperrors"
	"isp-gate-service/request"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/grpc"
	"github.com/txix-open/isp-kit/requestid"
	"golang.org/x/net/context"
)

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
	request.URL.Path = ctx.EndpointMeta().Endpoint
	setHttpHeaders(ctx, request.Header, p.skipAuth)

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

func setHttpHeaders(ctx *request.Context, header http.Header, skipAuth bool) {
	header.Set(requestid.Header, requestid.FromContext(ctx.Context()))
	if skipAuth {
		return
	}

	userAuthData, err := ctx.GetUserAuthData()
	if err == nil {
		for key, values := range userAuthData.ExtraHeaders {
			for _, value := range values {
				header.Add(key, value)
			}
		}
		header.Set(userAuthData.IdentityHeader, userAuthData.Identity)
	}

	appAuthData, err := ctx.GetAuthData()
	if err == nil {
		header.Set(grpc.SystemIdHeader, strconv.Itoa(appAuthData.SystemId))
		header.Set(grpc.DomainIdHeader, strconv.Itoa(appAuthData.DomainId))
		header.Set(grpc.ServiceIdHeader, strconv.Itoa(appAuthData.ServiceId))
		header.Set(grpc.ApplicationIdHeader, strconv.Itoa(appAuthData.ApplicationId))
		encodedAppName := base64.StdEncoding.EncodeToString([]byte(appAuthData.AppName))
		header.Set(grpc.ApplicationNameHeader, encodedAppName) //nolint:canonicalheader
	}

	if ctx.IsAdminAuthenticated() {
		header.Set(xAdminIdHeader, strconv.Itoa(ctx.AdminId())) //nolint:canonicalheader
	} else {
		header.Del(xAdminIdHeader) //nolint:canonicalheader
	}
}

func defaultTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return dialer.DialContext
}
