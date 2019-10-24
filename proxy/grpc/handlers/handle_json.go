package handlers

import (
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/config"
	isp "github.com/integration-system/isp-lib/proto/stubs"
	log "github.com/integration-system/isp-log"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"isp-gate-service/conf"
	"isp-gate-service/journal"
	"isp-gate-service/log_code"
	"isp-gate-service/service"
	"isp-gate-service/utils"
	"time"
)

var handleJson handleJsonDesc

type handleJsonDesc struct{}

func (p handleJsonDesc) Complete(c *fasthttp.RequestCtx, method string, client *backend.RxGrpcClient) {
	//body, err := utils.ReadJsonBody(c)
	body := c.Request.Body()
	/*if err != nil {
		streaming.LogError(log_code.TypeData.JsonContent, method, err)
		streaming.SendError(err.Error(), codes.InvalidArgument, []interface{}{err.Error()}, c)
		return
	}*/

	md, methodName := makeMetadata(&c.Request.Header, method)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	cfg := config.GetRemote().(*conf.RemoteConfig)
	ctx, cancel := context.WithTimeout(ctx, cfg.GetSyncInvokeTimeout())
	defer cancel()

	cli, err := client.Conn()
	if err != nil {
		logHandlerError(log_code.TypeData.JsonContent, methodName, err)
		utils.SendError(errorMsgInternal, codes.Internal, []interface{}{err.Error()}, c)
		return
	}

	//structBody := u.ConvertInterfaceToGrpcStruct(body)
	currentTime := time.Now()
	response, invokerErr := cli.Request(
		ctx,
		&isp.Message{
			Body: &isp.Message_BytesBody{BytesBody: body},
		},
	)
	service.Metrics.UpdateRouterResponseTime(time.Since(currentTime) / 1e6)

	if data, status, err := getResponse(response, invokerErr); err == nil {
		c.SetStatusCode(status)
		_, _ = c.Write(data)
		if cfg.Journal.Enable && service.JournalMethodsMatcher.Match(methodName) {
			if invokerErr != nil {
				if err := journal.Client.Error(methodName, body, data, invokerErr); err != nil {
					log.Warnf(log_code.WarnJournalCouldNotWriteToFile, "could not write to file journal: %v", err)
				}
			} else {
				if err := journal.Client.Info(methodName, body, data); err != nil {
					log.Warnf(log_code.WarnJournalCouldNotWriteToFile, "could not write to file journal: %v", err)
				}
			}
		}
	} else {
		logHandlerError(log_code.TypeData.JsonContent, methodName, err)
		utils.SendError(errorMsgInternal, codes.Internal, []interface{}{err.Error()}, c)
	}
}
