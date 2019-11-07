package handler

import (
	"github.com/integration-system/isp-lib/config"
	log "github.com/integration-system/isp-log"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/approve"
	"isp-gate-service/authenticate"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/journal"
	"isp-gate-service/log_code"
	"isp-gate-service/proxy"
	"isp-gate-service/proxy/response"
	"isp-gate-service/service"
	"net/http"
	"time"
)

var helper handlerHelper

type handlerHelper struct{}

func CompleteRequest(ctx *fasthttp.RequestCtx) {
	currentTime := time.Now()
	uri := string(ctx.RequestURI())

	resp := helper.AuthenticateApproveProxy(ctx)

	executionTime := time.Since(currentTime) / 1e6

	statusCode := ctx.Response.StatusCode()
	service.Metrics.UpdateStatusCounter(helper.SetMetricStatus(statusCode))
	if statusCode == http.StatusOK {
		service.Metrics.UpdateResponseTime(executionTime)
		service.Metrics.UpdateMethodResponseTime(uri, executionTime)
	}

	if config.GetRemote().(*conf.RemoteConfig).Journal.Enable && service.JournalMethodsMatcher.Match(uri) {
		if resp.Error != nil {
			if err := journal.Client.Error(uri, resp.RequestBody, resp.ResponseBody, resp.Error); err != nil {
				log.Warnf(log_code.WarnJournalCouldNotWriteToFile, "could not write to file journal: %v", err)
			}
		} else {
			if err := journal.Client.Info(uri, resp.RequestBody, resp.ResponseBody); err != nil {
				log.Warnf(log_code.WarnJournalCouldNotWriteToFile, "could not write to file journal: %v", err)
			}
		}
	}
}

func (handlerHelper) AuthenticateApproveProxy(ctx *fasthttp.RequestCtx) domain.ProxyResponse {
	path := string(ctx.Path())

	applicationId, err := authenticate.Do(ctx)
	if err != nil {
		status := codes.Unknown
		switch e := err.(type) {
		case authenticate.ErrorDescription:
			status = e.ConvertToGrpcStatus()
		}
		return response.Create(ctx, response.Option.SetAndSendError("unauthorized", status, err))
	}

	if approver := approve.GetApprove(applicationId); approver != nil && !approver.ApproveMethod(path) {
		err := errors.New("approve error")
		return response.Create(ctx, response.Option.SetAndSendError("forbidden", codes.PermissionDenied, err))
	}

	p := proxy.Find(path)
	if p != nil {
		return p.ProxyRequest(ctx)
	} else {
		err := errors.Errorf("unknown path %s", path)
		return response.Create(ctx, response.Option.SetAndSendError("not found", codes.NotFound, err))
	}
}

func (handlerHelper) SetMetricStatus(statusCode int) string {
	metricStatus := "5xx"
	switch {
	case statusCode >= 100 && statusCode < 200:
		metricStatus = "1xx"
	case statusCode >= 200 && statusCode < 300:
		metricStatus = "2xx"
	case statusCode >= 300 && statusCode < 400:
		metricStatus = "3xx"
	case statusCode >= 400 && statusCode < 500:
		metricStatus = "4xx"
	case statusCode >= 500 && statusCode < 600:
		metricStatus = "5xx"
	}
	return metricStatus
}

//
//func (h handlerHelper) CheckPath(path []byte) (bool, bool) {
//	path = h.getPathWithoutPrefix(path)
//	_, implemented := routing.AddressMap[string(path)]
//	_, inner := routing.InnerAddressMap[string(path)]
//	return implemented, inner
//}
//
//func (handlerHelper) getPathWithoutPrefix(path []byte) []byte {
//	firstFound := false
//	for i, value := range path {
//		if value == '/' {
//			if firstFound {
//				return path[i+1:]
//			} else {
//				firstFound = true
//			}
//		}
//	}
//	return []byte{}
//}
