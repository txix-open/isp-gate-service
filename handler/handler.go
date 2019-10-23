package handler

import (
	"encoding/json"
	"github.com/integration-system/isp-lib/http"
	"github.com/integration-system/isp-lib/structure"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/authenticate"
	"isp-gate-service/proxy"
)

func Complete(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())

	if err := authenticate.Compete(ctx); err != nil {
		statusCode := codes.Unauthenticated
		response := structure.GrpcError{
			ErrorMessage: "unauthenticated", ErrorCode: statusCode.String(),
			Details: []interface{}{err},
		}
		ctx.Response.Header.SetContentType("application/json; charset=utf-8")
		ctx.SetStatusCode(http.CodeToHttpStatus(statusCode))
		message, _ := json.Marshal(response)
		_, _ = ctx.Write(message)
		return
	}

	p := proxy.Find(path)
	if p != nil {
		p.ProxyRequest(ctx)
	} else {
		statusCode := codes.NotFound
		response := structure.GrpcError{
			ErrorMessage: "unknown path", ErrorCode: statusCode.String(),
			Details: []interface{}{map[string]string{"path": path}},
		}
		ctx.Response.Header.SetContentType("application/json; charset=utf-8")
		ctx.SetStatusCode(http.CodeToHttpStatus(statusCode))
		message, _ := json.Marshal(response)
		_, _ = ctx.Write(message)
	}
}
