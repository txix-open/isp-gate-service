package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/integration-system/isp-kit/log"
	"github.com/integration-system/isp-lib/v2/config"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/accounting"
	"isp-gate-service/authenticate"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/proxy"
	"isp-gate-service/routing"
	"isp-gate-service/service"
	"isp-gate-service/service/matcher"
	"isp-gate-service/utils"
)

const (
	execution = 1e6
)

var errAccounting = errors.New("accounting error")

type Handler struct {
	logger log.Logger
}

func New(logger log.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

func (h Handler) CompleteRequest(ctx *fasthttp.RequestCtx) {
	currentTime := time.Now()

	appId, adminId, path, resp := h.authenticateAccountingProxy(ctx)
	executionTime := time.Since(currentTime) / execution

	statusCode := ctx.Response.StatusCode()
	service.Metrics.UpdateStatusCounter(h.setMetricStatus(statusCode))
	if statusCode == http.StatusOK {
		service.Metrics.UpdateMethodResponseTime(path, executionTime)
	}

	logEnable := config.GetRemote().(*conf.RemoteConfig).JournalSetting.Journal.Enable
	// nolintlint
	if logEnable && matcher.JournalMethods.Match(path) {
		requestBody, responseBody, err := resp.Get()
		fields := []log.Field{
			log.ByteString("http_method", ctx.Method()),
			log.String("remote_addr", ctx.RemoteAddr().String()),
			log.ByteString("x_forwarded_for", ctx.Request.Header.Peek("X-Forwarded-For")),
			log.Int("status_code", ctx.Response.StatusCode()),
			log.String("path", path),
			log.Int32("application_id", appId),
			log.Int64("admin_id", adminId),
			log.ByteString("request", requestBody),
			log.ByteString("response", responseBody),
		}
		if err != nil {
			fields = append(fields, log.Any("error", err))
			h.logger.Error(ctx, "unsuccessful request", fields...)
		} else {
			h.logger.Info(ctx, "successful request", fields...)
		}
	}
}

func (h Handler) authenticateAccountingProxy(ctx *fasthttp.RequestCtx) (int32, int64, string, domain.ProxyResponse) {
	var (
		appId       int32 = -1
		adminId     int64 = -1
		initialPath       = string(ctx.Path())
	)

	p, path := proxy.Find(initialPath)
	if p == nil {
		msg := fmt.Sprintf("unknown proxy for '%s'", initialPath)
		utils.WriteError(ctx, msg, codes.NotFound, nil)
		return appId, adminId, initialPath, domain.Create().SetError(errors.New(msg))
	}

	if !p.SkipExistCheck() {
		if _, ok := routing.AllMethods[path]; !ok {
			msg := "not implemented"
			utils.WriteError(ctx, msg, codes.Unimplemented, nil)
			return appId, adminId, path, domain.Create().SetError(errors.New(msg))
		}
	}

	if !p.SkipAuth() {
		var err error
		appId, adminId, err = authenticate.Do(ctx, path)
		if err != nil {
			message := "unknown error"
			status := codes.Unknown
			details := make([]interface{}, 0)
			if e, ok := err.(authenticate.ErrorDescription); ok {
				message = e.Message()
				status = e.ConvertToGrpcStatus()
				details = e.Details()
			}
			utils.WriteError(ctx, message, status, details)
			return appId, adminId, path, domain.Create().SetError(err)
		}

		if !accounting.AcceptRequest(appId, path) {
			err := errAccounting
			utils.WriteError(ctx, "too many requests", codes.ResourceExhausted, nil)
			return appId, adminId, path, domain.Create().SetError(err)
		}
	}

	return appId, adminId, path, p.ProxyRequest(ctx, path)
}

func (Handler) setMetricStatus(statusCode int) string {
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
