package assembly

import (
	"net/http"
	"time"

	"isp-gate-service/conf"
	"isp-gate-service/middleware"
	"isp-gate-service/proxy"
	"isp-gate-service/repository"
	"isp-gate-service/routes"
	"isp-gate-service/service"

	mux2 "github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/grpc/client"
	"github.com/txix-open/isp-kit/lb"
	"github.com/txix-open/isp-kit/log"
)

type Locator struct {
	logger                      log.Logger
	grpcClientByModuleName      map[string]*client.Client
	httpHostManagerByModuleName map[string]*lb.RoundRobin
	routes                      *routes.Routes
	systemCli                   *client.Client
	adminCli                    *client.Client
	lockerCli                   *client.Client
}

func NewLocator(
	logger log.Logger,
	grpcClientByModuleName map[string]*client.Client,
	httpHostManagerByModuleName map[string]*lb.RoundRobin,
	routes *routes.Routes,
	systemCli *client.Client,
	adminCli *client.Client,
	lockerCli *client.Client,
) Locator {
	return Locator{
		logger:                      logger,
		grpcClientByModuleName:      grpcClientByModuleName,
		httpHostManagerByModuleName: httpHostManagerByModuleName,
		routes:                      routes,
		systemCli:                   systemCli,
		adminCli:                    adminCli,
		lockerCli:                   lockerCli,
	}
}

func (l Locator) Handler(config conf.Remote, locations []conf.Location) (http.Handler, error) { // nolint:funlen
	systemRepo := repository.NewSystem(l.systemCli)
	adminRepo := repository.NewAdmin(l.adminCli)

	authenticationCache := repository.NewAuthenticationCache(time.Duration(config.Caching.AuthenticationDataInSec) * time.Second)
	authentication := service.NewAuthentication(authenticationCache, systemRepo)

	adminService := service.NewAdmin(
		repository.NewAuthorizationCache(time.Duration(config.Caching.AuthorizationDataInSec)*time.Second),
		adminRepo,
	)

	authorization := service.NewAuthorization(
		repository.NewAuthorizationCache(time.Duration(config.Caching.AuthorizationDataInSec)*time.Second),
		systemRepo,
	)

	lockRepo := repository.NewLocker(l.lockerCli)
	dailyLimitService := service.NewDailyLimit(lockRepo, config.DailyLimits)
	throttlingService := service.NewThrottling(lockRepo, config.Throttling)

	mux := mux2.NewRouter()
	for _, location := range locations {
		var proxyFunc middleware.Handler
		enableBodyLog := config.Logging.BodyLogEnable

		switch location.Protocol {
		case conf.GrpcProtocol:
			cli := l.grpcClientByModuleName[location.TargetModule]
			proxyFunc = proxy.NewGrpc(cli, location.SkipAuth, time.Duration(config.Http.ProxyTimeoutInSec)*time.Second)
		case conf.HttpProtocol:
			hostManager := l.httpHostManagerByModuleName[location.TargetModule]
			proxyFunc = proxy.NewHttp(hostManager, location.SkipAuth, time.Duration(config.Http.ProxyTimeoutInSec)*time.Second)
		case conf.WsProtocol:
			hostManager := l.httpHostManagerByModuleName[location.TargetModule]
			proxyFunc = proxy.NewWs(hostManager, location.SkipAuth)
			enableBodyLog = false
		default:
			return nil, errors.Errorf("not supported protocol %s", location.Protocol)
		}

		forwardReqIdByAppId := make(map[int]bool, len(config.ForwardReqIdClientSettings))
		for _, setting := range config.ForwardReqIdClientSettings {
			forwardReqIdByAppId[setting.ApplicationId] = setting.ForwardRequestId
		}

		handler := middleware.Chain(
			proxyFunc,
			middleware.Logger(
				l.logger, config.Logging.RequestLogEnable,
				enableBodyLog,
				config.Logging.SkipBodyLoggingEndpointPrefixes,
				config.Logging.EnableForceUnescapingUnicode,
			),
			middleware.ErrorHandler(l.logger),
			middleware.Authenticate(authentication),
			middleware.AdminAuthenticate(adminService),
			middleware.Authorize(authorization, l.logger),
			middleware.AdminAuthorize(l.routes, adminService),
			middleware.Throttling(throttlingService),
			middleware.DailyLimit(dailyLimitService),
			middleware.RequestId(config.EnableClientRequestIdForwarding, forwardReqIdByAppId),
		)
		if location.SkipAuth {
			handler = middleware.Chain(
				proxyFunc,
				middleware.Logger(l.logger, config.Logging.RequestLogEnable,
					enableBodyLog,
					config.Logging.SkipBodyLoggingEndpointPrefixes,
					config.Logging.EnableForceUnescapingUnicode,
				),
				middleware.ErrorHandler(l.logger),
				middleware.RequestId(config.EnableClientRequestIdForwarding, forwardReqIdByAppId),
			)
		}
		entrypoint := middleware.Entrypoint(
			config.Http.MaxRequestBodySizeInMb*1024*1024, //nolint:mnd
			handler,
			middleware.EntryPointConfig{
				WithPrefix: location.WithPrefix,
				PathPrefix: location.PathPrefix,
			},
			l.routes,
			l.logger,
		)
		mux.PathPrefix(location.PathPrefix).Handler(entrypoint)
	}

	return mux, nil
}
