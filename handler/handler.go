package handler

import (
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/approve"
	"isp-gate-service/authenticate"
	"isp-gate-service/proxy"
	"isp-gate-service/utils"
)

func Complete(ctx *fasthttp.RequestCtx) {
	applicationId, err := authenticate.Do(ctx)
	if err != nil {
		status := codes.Unknown
		switch e := err.(type) {
		case authenticate.ErrorDescription:
			status = e.ConvertToGrpcStatus()
		}
		utils.SendError("authenticate error", status, nil, ctx)
		return
	}

	path := string(ctx.Path())

	if !approve.Complete(applicationId, path) {
		utils.SendError("approve error", codes.PermissionDenied, nil, ctx)
		return
	}

	p := proxy.Find(path)
	if p != nil {
		p.ProxyRequest(ctx)
	} else {
		utils.SendError("unknown path", codes.NotFound, []interface{}{map[string]string{"path": path}}, ctx)
	}
}
