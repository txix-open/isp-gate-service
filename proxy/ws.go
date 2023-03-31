package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/tomakado/websocketproxy"
	"isp-gate-service/httperrors"
	"isp-gate-service/request"
)

type Ws struct {
	hostManager HttpHostManager
	skipAuth    bool
}

func NewWs(hostManager HttpHostManager, skipAuth bool) Ws {
	return Ws{
		hostManager: hostManager,
		skipAuth:    skipAuth,
	}
}

//nolint:gomnd
func (ws Ws) Handle(ctx *request.Context) error {
	host, err := ws.hostManager.Next()
	if err != nil {
		return errors.WithMessage(err, "ws: next host")
	}

	rawUrl := fmt.Sprintf("ws://%s", host)
	target, err := url.Parse(rawUrl)
	if err != nil {
		return errors.WithMessage(err, "ws: parse url")
	}

	request := ctx.Request()
	request.URL.Path = ctx.Endpoint()

	var resultError error
	proxy := websocketproxy.NewProxy(target)
	proxy.Director = func(incoming *http.Request, out http.Header) {
		_ = setHttpHeaders(ctx, out, ws.skipAuth)
	}
	proxy.Upgrader = &websocket.Upgrader{
		HandshakeTimeout: 5 * time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		WriteBufferPool:  nil,
		Subprotocols:     nil,
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			resultError = httperrors.New(
				http.StatusServiceUnavailable,
				"upstream is not available",
				errors.WithMessagef(reason, "ws proxy to %s", host),
			)
		},
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: false,
	}

	proxy.ServeHTTP(ctx.ResponseWriter(), ctx.Request())

	return resultError
}
