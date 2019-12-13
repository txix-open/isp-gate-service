package websocket

import (
	"errors"
	"github.com/fasthttp/websocket"
	"github.com/integration-system/isp-lib/structure"
	log "github.com/integration-system/isp-log"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"io"
	"isp-gate-service/domain"
	"isp-gate-service/log_code"
	"isp-gate-service/utils"
	"net"
	"net/http"
)

const (
	writeBufSize = 4 << 10
	readBufSize  = 4 << 10
)

// used to filter client request headers
var forbiddenDuplicateHeaders = map[string]struct{}{
	"Upgrade":                  {},
	"Connection":               {},
	"Sec-Websocket-Key":        {},
	"Sec-Websocket-Version":    {},
	"Sec-Websocket-Extensions": {},
	"Sec-Websocket-Protocol":   {},
}

var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  readBufSize,
	WriteBufferSize: writeBufSize,
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
}

type websocketProxy struct {
	addrs    *RoundRobinAddrs
	skipAuth bool
}

func NewProxy(skipAuth bool) *websocketProxy {
	return &websocketProxy{addrs: nil, skipAuth: skipAuth}
}

func (p *websocketProxy) Consumer(addressList []structure.AddressConfiguration) bool {
	if len(addressList) == 0 {
		p.addrs = nil
	} else {
		p.addrs = NewRoundRobinAddrs(addressList)
	}
	return true
}

func (p *websocketProxy) ProxyRequest(ctx *fasthttp.RequestCtx, path string) domain.ProxyResponse {
	addrs := p.addrs
	if addrs == nil {
		msg := "no available address"
		log.Error(log_code.ErrorWebsocketProxy, msg)
		utils.WriteError(ctx, msg, codes.Internal, nil)
		return domain.Create().
			SetRequestBody(ctx.Request.Body()).
			SetResponseBody(ctx.Response.Body()).
			SetError(errors.New(msg))
	}

	reqHeaders := fasthttp.RequestHeader{}
	ctx.Request.Header.CopyTo(&reqHeaders)

	if addr, _, err := net.SplitHostPort(ctx.RemoteAddr().String()); err == nil {
		reqHeaders.Add("X-Forwarded-For", addr)
	}

	err := upgrader.Upgrade(ctx, func(incomingConn *websocket.Conn) {
		outgoingDialer := websocket.Dialer{
			ReadBufferSize:  readBufSize,
			WriteBufferSize: writeBufSize,
			NetDial: func(network, address string) (net.Conn, error) {
				return net.Dial(network, address)
			},
		}

		addr := addrs.Get()
		url := "ws://" + addr.GetAddress() + "/" + path
		header := http.Header{}

		reqHeaders.VisitAll(func(key, value []byte) {
			keyStr := string(key)
			if _, forbidden := forbiddenDuplicateHeaders[keyStr]; !forbidden {
				header.Add(keyStr, string(value))
			}
		})

		outgoingConn, _, err := outgoingDialer.Dial(url, header)
		if err == nil {
			go func() {
				_ = p.proxyConn(outgoingConn, incomingConn)
			}()
			_ = p.proxyConn(incomingConn, outgoingConn)
		} else {
			log.Errorf(log_code.ErrorWebsocketProxy, "unable to connect to service %s: %v", url, err)
			_ = incomingConn.Close()
		}
	})

	return domain.Create().
		SetRequestBody(ctx.Request.Body()).
		SetResponseBody(ctx.Response.Body()).
		SetError(err)
}

func (p *websocketProxy) SkipAuth() bool {
	return p.skipAuth
}

// no-op
func (p *websocketProxy) Close() {
}

func (p *websocketProxy) proxyConn(from, to *websocket.Conn) error {
	defer func() {
		_ = from.Close()
		_ = to.Close()
	}()
	for {
		msgType, reader, err := from.NextReader()
		if err != nil {
			return err
		}
		writer, err := to.NextWriter(msgType)
		if err != nil {
			return err
		}
		if _, err := io.Copy(writer, reader); err != nil {
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}
	}
}
