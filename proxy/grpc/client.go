package grpc

import (
	"github.com/integration-system/isp-lib/v2/backend"
	"github.com/integration-system/isp-lib/v2/structure"
	log "github.com/integration-system/isp-log"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/log_code"
	"isp-gate-service/proxy/grpc/handlers"
	"isp-gate-service/utils"
)

type grpcProxy struct {
	client         *backend.RxGrpcClient
	skipAuth       bool
	skipExistCheck bool
}

func NewProxy(skipAuth, skipExistCheck bool) *grpcProxy {
	return &grpcProxy{
		client: backend.NewRxGrpcClient(
			backend.WithDialOptions(
				grpc.WithInsecure(), grpc.WithBlock(),
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(conf.DefaultMaxResponseBodySize))),
				grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(int(conf.DefaultMaxResponseBodySize))),
			),
			backend.WithDialingErrorHandler(func(err error) {
				log.Errorf(log_code.ErrorClientGrpc, "dialing err: %v", err)
			})),
		skipAuth:       skipAuth,
		skipExistCheck: skipExistCheck,
	}
}

func (p *grpcProxy) ProxyRequest(ctx *fasthttp.RequestCtx, path string) domain.ProxyResponse {
	if p.client.InternalGrpcClient == nil {
		msg := "client undefined"
		log.Error(log_code.ErrorClientGrpc, msg)
		utils.WriteError(ctx, msg, codes.Internal, nil)
		return domain.Create().
			SetRequestBody(ctx.Request.Body()).
			SetResponseBody(ctx.Response.Body()).
			SetError(errors.New(msg))
	}

	return handlers.Handler.Get(ctx).Complete(ctx, path, p.client)
}

func (p *grpcProxy) Consumer(addr []structure.AddressConfiguration) bool {
	return p.client.ReceiveAddressList(addr)
}

func (p *grpcProxy) SkipAuth() bool {
	return p.skipAuth
}

func (p *grpcProxy) SkipExistCheck() bool {
	return p.skipExistCheck
}

func (p *grpcProxy) Close() {
	p.client.Close()
}
