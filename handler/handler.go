package handler

import (
	"github.com/integration-system/isp-lib/config"
	log "github.com/integration-system/isp-log"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/accounting"
	"isp-gate-service/authenticate"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/journal"
	"isp-gate-service/log_code"
	"isp-gate-service/proxy"
	"isp-gate-service/service"
	"isp-gate-service/service/matcher"
	"isp-gate-service/utils"
	"net/http"
	"time"
)

var helper handlerHelper

type handlerHelper struct{}

func CompleteRequest(ctx *fasthttp.RequestCtx) {
	currentTime := time.Now()
	uri := string(ctx.RequestURI())

	resp := helper.AuthenticateAccountingProxy(ctx)

	executionTime := time.Since(currentTime) / 1e6

	statusCode := ctx.Response.StatusCode()
	service.Metrics.UpdateStatusCounter(helper.SetMetricStatus(statusCode))
	if statusCode == http.StatusOK {
		service.Metrics.UpdateMethodResponseTime(uri, executionTime)
	}

	requestBody, responseBody, err := resp.Get()
	if config.GetRemote().(*conf.RemoteConfig).JournalSetting.Journal.Enable {
		if matcher.JournalMethods.Match(uri) {
			if err != nil {
				if err := journal.Client.Error(uri, requestBody, responseBody, err); err != nil {
					log.Warnf(log_code.WarnJournalCouldNotWriteToFile, "could not write to file journal: %v", err)
				}
			} else {
				if err := journal.Client.Info(uri, requestBody, responseBody); err != nil {
					log.Warnf(log_code.WarnJournalCouldNotWriteToFile, "could not write to file journal: %v", err)
				}
			}
		}
	}
}

func (handlerHelper) AuthenticateAccountingProxy(ctx *fasthttp.RequestCtx) domain.ProxyResponse {
	path := string(ctx.Path())

	p := proxy.Find(path)
	if p == nil {
		err := errors.Errorf("unknown path %s", path)
		utils.WriteError(ctx, "not found", codes.NotFound, nil)
		return domain.Create().SetError(err)
	}

	applicationId, err := authenticate.Do(ctx)
	if err != nil {
		status := codes.Unknown
		details := make([]interface{}, 0)
		switch e := err.(type) {
		case authenticate.ErrorDescription:
			status = e.ConvertToGrpcStatus()
			details = e.Details()
		}
		utils.WriteError(ctx, "unauthorized", status, details)
		return domain.Create().SetError(err)
	}

	if account := accounting.GetAccounting(applicationId); account != nil && !account.Accept(path) {
		err := errors.New("accounting error")
		utils.WriteError(ctx, "forbidden", codes.ResourceExhausted, nil)
		return domain.Create().SetError(err)
	}

	return p.ProxyRequest(ctx)
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
