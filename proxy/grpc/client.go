package grpc

import (
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/structure"
	log "github.com/integration-system/isp-log"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/log_code"
	"isp-gate-service/proxy/grpc/handlers"
	"isp-gate-service/proxy/response"
)

type grpcProxy struct {
	client *backend.RxGrpcClient
}

func NewProxy() *grpcProxy {
	return &grpcProxy{client: backend.NewRxGrpcClient(
		backend.WithDialOptions(
			grpc.WithInsecure(), grpc.WithBlock(),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(conf.DefaultMaxResponseBodySize))),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(int(conf.DefaultMaxResponseBodySize))),
		),
		backend.WithDialingErrorHandler(func(err error) {
			log.Errorf(log_code.ErrorClientGrpc, "dialing err: %v", err)
		}),
	)}
}

func (p *grpcProxy) ProxyRequest(ctx *fasthttp.RequestCtx) domain.ProxyResponse {
	if p.client.InternalGrpcClient == nil {
		msg := "client undefined"
		log.Error(log_code.ErrorClientGrpc, msg)
		return response.Create(ctx, response.Option.SetAndSendError(msg, codes.Internal, errors.New(msg)))
	}

	uri := string(ctx.RequestURI())
	return handlers.Handler.Get(ctx).Complete(ctx, uri, p.client)
}

func (p *grpcProxy) Consumer(addr []structure.AddressConfiguration) bool {
	return p.client.ReceiveAddressList(addr)
}

func (p *grpcProxy) Close() {
	p.client.Close()
}
