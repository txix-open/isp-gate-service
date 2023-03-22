package assembly

import (
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"isp-gate-service/conf"
	"isp-gate-service/middleware"
	"isp-gate-service/proxy"
	"isp-gate-service/repository"
	"isp-gate-service/routes"
	"isp-gate-service/service"

	"github.com/integration-system/isp-kit/grpc/client"
	"github.com/integration-system/isp-kit/lb"
	"github.com/integration-system/isp-kit/log"
	"github.com/pkg/errors"
)

type Locator struct {
	logger                      log.Logger
	grpcClientByModuleName      map[string]*client.Client
	httpHostManagerByModuleName map[string]*lb.RoundRobin
	routes                      *routes.Routes
	systemCli                   *client.Client
	adminCli                    *client.Client
}

func NewLocator(
	logger log.Logger,
	grpcClientByModuleName map[string]*client.Client,
	httpHostManagerByModuleName map[string]*lb.RoundRobin,
	routes *routes.Routes,
	systemCli *client.Client,
	adminCli *client.Client,
) Locator {
	return Locator{
		logger:                      logger,
		grpcClientByModuleName:      grpcClientByModuleName,
		httpHostManagerByModuleName: httpHostManagerByModuleName,
		routes:                      routes,
		systemCli:                   systemCli,
		adminCli:                    adminCli,
	}
}

func (l Locator) Handler(config conf.Remote, locations []conf.Location, redisCli redis.UniversalClient) (http.Handler, error) {
	systemRepo := repository.NewSystem(l.systemCli)
	adminRepo := repository.NewAdmin(l.adminCli)

	authenticationCache := repository.NewAuthenticationCache(time.Duration(config.Caching.AuthenticationDataInSec) * time.Second)
	authentication := service.NewAuthentication(authenticationCache, systemRepo)

	adminService := service.NewAdmin(adminRepo)

	authorizationCache := repository.NewAuthorizationCache(time.Duration(config.Caching.AuthorizationDataInSec) * time.Second)
	authorization := service.NewAuthorization(authorizationCache, systemRepo)

	dailyLimitRepo := repository.NewDailyLimit(redisCli)
	dailyLimitService := service.NewDailyLimit(dailyLimitRepo, config.DailyLimits)

	throttlingRepo := repository.NewThrottling(redisCli)
	throttlingService := service.NewThrottling(throttlingRepo, config.Throttling)

	enableBodyLog := config.Logging.BodyLogEnable

	mux := http.NewServeMux()
	for _, location := range locations {
		var proxyFunc middleware.Handler
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

		handler := middleware.Chain(
			proxyFunc,
			middleware.RequestId(),
			middleware.Logger(l.logger, config.Logging.RequestLogEnable, enableBodyLog),
			middleware.ErrorHandler(l.logger),
			middleware.Authenticate(authentication),
			middleware.AdminAuthenticate(adminService),
			middleware.Authorize(authorization, l.logger),
			middleware.AdminAuthorize(l.routes),
			middleware.Throttling(throttlingService),
			middleware.DailyLimit(dailyLimitService),
		)
		if location.SkipAuth {
			handler = middleware.Chain(
				proxyFunc,
				middleware.RequestId(),
				middleware.Logger(l.logger, config.Logging.RequestLogEnable, enableBodyLog),
				middleware.ErrorHandler(l.logger),
			)
		}

		entrypoint := middleware.Entrypoint(
			config.Http.MaxRequestBodySizeInMb*1024*1024, //nolint:gomnd
			handler,
			location.PathPrefix,
			l.logger,
		)
		mux.Handle(fmt.Sprintf("%s/", location.PathPrefix), entrypoint)
	}

	return mux, nil
}
