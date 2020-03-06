package handlers

import (
	"github.com/integration-system/isp-lib/v2/backend"
	"github.com/integration-system/isp-lib/v2/config"
	isp "github.com/integration-system/isp-lib/v2/proto/stubs"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/log_code"
	"isp-gate-service/utils"
)

var handleJson handleJsonDesc

type handleJsonDesc struct{}

func (p handleJsonDesc) Complete(c *fasthttp.RequestCtx, method string, client *backend.RxGrpcClient) domain.ProxyResponse {
	body := c.Request.Body()

	md, methodName := makeMetadata(&c.Request.Header, method)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	grpcSetting := config.GetRemote().(*conf.RemoteConfig).GrpcSetting
	ctx, cancel := context.WithTimeout(ctx, grpcSetting.GetSyncInvokeTimeout())
	defer cancel()

	cli, err := client.Conn()
	if err != nil {
		logHandlerError(log_code.TypeData.JsonContent, methodName, err)
		utils.WriteError(c, errorMsgInternal, codes.Internal, nil)
		return domain.Create().SetError(err)
	}

	msg, invokerErr := cli.Request(
		ctx,
		&isp.Message{
			Body: &isp.Message_BytesBody{BytesBody: body},
		},
	)

	if response, status, err := getResponse(msg, invokerErr); err == nil {
		c.SetStatusCode(status)
		_, _ = c.Write(response)
		return domain.Create().SetError(invokerErr)
	} else {
		logHandlerError(log_code.TypeData.JsonContent, methodName, err)
		utils.WriteError(c, errorMsgInternal, codes.Internal, nil)
		return domain.Create().SetError(err)
	}
}
