package assembly

import (
	"fmt"
	"net/http"

	"github.com/integration-system/isp-kit/grpc/client"
	"github.com/integration-system/isp-kit/http/endpoint"
	"github.com/integration-system/isp-kit/json"
	"github.com/integration-system/isp-kit/lb"
	"github.com/integration-system/isp-kit/log"
	"github.com/integration-system/isp-kit/requestid"
	"github.com/pkg/errors"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/middleware"
	"isp-gate-service/proxy"
	"isp-gate-service/routes"
	"isp-gate-service/service"
)

type Locator struct {
	logger                      endpoint.Logger
	grpcClientByModuleName      map[string]*client.Client
	httpHostManagerByModuleName map[string]*lb.RoundRobin
	routes                      *routes.Routes
}

func NewLocator(
	logger endpoint.Logger,
	grpcClientByModuleName map[string]*client.Client,
	httpHostManagerByModuleName map[string]*lb.RoundRobin,
	routes *routes.Routes,
) Locator {
	return Locator{
		logger:                      logger,
		grpcClientByModuleName:      grpcClientByModuleName,
		httpHostManagerByModuleName: httpHostManagerByModuleName,
		routes:                      routes,
	}
}

func (l Locator) Handler(config conf.Remote, locations []conf.Location) (http.Handler, error) {
	adminService := service.NewAdmin(config.TokensSetting.AdminSecret, l.routes)

	mux := http.NewServeMux()
	for _, location := range locations {
		var proxyFunc middleware.Handler
		switch location.Protocol {
		case conf.GrpcProtocol:
			cli := l.grpcClientByModuleName[location.TargetModule]
			proxyFunc = proxy.NewGrpc(cli)
		case conf.HttpProtocol:
			hostManager := l.httpHostManagerByModuleName[location.TargetModule]
			proxyFunc = proxy.NewHttp(hostManager)
		case conf.WebsocketProtocol:

		default:
			return nil, errors.New("")
		}

		middlewares := []middleware.Middleware{
			middleware.Path(location.PathPrefix),
			middleware.Authenticate(adminService),
			middleware.Authorize(adminService),
		}
		for i := len(middlewares) - 1; i >= 0; i-- {
			proxyFunc = middlewares[i](proxyFunc)
		}

		var adapter = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := &middleware.Context{
				Id:             requestid.Next(),
				Request:        request,
				ResponseWriter: writer,
			}
			err := proxyFunc.Handle(ctx)
			if err != nil {
				l.handleError(ctx, err)
			} else {
				l.logger.Info(request.Context(), "successful request",
					log.String("id", ctx.Id),
					log.String("path", ctx.Path),
					log.Int("application_id", ctx.AppId),
					log.Int("admin_id", ctx.AdminId),
				)
			}
		})
		mux.HandleFunc(fmt.Sprintf("%s/", location.PathPrefix), adapter)
	}

	return mux, nil
}

func (l Locator) handleError(ctx *middleware.Context, err error) {
	l.logger.Error(ctx.Request.Context(), "unsuccessful request",
		log.String("id", ctx.Id),
		log.Any("error", err),
	)

	status := 0
	details := make(map[string]string)
	switch {
	case errors.Is(err, domain.ErrAuthorize):
		status = http.StatusUnauthorized
		details["message"] = domain.ErrAuthorize.Error()
	case errors.Is(err, domain.ErrAuthenticate):
		status = http.StatusUnauthorized
		details["message"] = domain.ErrAuthenticate.Error()
	default:
		status = http.StatusInternalServerError
	}
	details["id"] = ctx.Id

	result := domain.Error{
		ErrorMessage: http.StatusText(status),
		ErrorCode:    http.StatusText(status),
		Details:      []interface{}{details},
	}

	body, err := json.Marshal(result)
	if err != nil {
		l.logger.Error(ctx.Request.Context(), "json error marshal",
			log.String("id", ctx.Id),
			log.Any("error", err),
		)
		return
	}

	ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.ResponseWriter.WriteHeader(status)
	_, err = ctx.ResponseWriter.Write(body)
	if err != nil {
		l.logger.Error(ctx.Request.Context(), "write error response",
			log.String("id", ctx.Id),
			log.Any("error", err),
		)
		return
	}
}
