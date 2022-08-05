package assembly

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/integration-system/isp-kit/grpc/client"
	"github.com/integration-system/isp-kit/lb"
	"github.com/integration-system/isp-kit/log"
	"github.com/pkg/errors"
	"isp-gate-service/conf"
	"isp-gate-service/middleware"
	"isp-gate-service/proxy"
	"isp-gate-service/repository"
	"isp-gate-service/routes"
	"isp-gate-service/service"
)

type Locator struct {
	logger                      log.Logger
	grpcClientByModuleName      map[string]*client.Client
	httpHostManagerByModuleName map[string]*lb.RoundRobin
	routes                      *routes.Routes
	systemCli                   *client.Client
}

func NewLocator(
	logger log.Logger,
	grpcClientByModuleName map[string]*client.Client,
	httpHostManagerByModuleName map[string]*lb.RoundRobin,
	routes *routes.Routes,
	systemCli *client.Client,
) Locator {
	return Locator{
		logger:                      logger,
		grpcClientByModuleName:      grpcClientByModuleName,
		httpHostManagerByModuleName: httpHostManagerByModuleName,
		routes:                      routes,
		systemCli:                   systemCli,
	}
}

func (l Locator) Handler(config conf.Remote, locations []conf.Location, redisCli redis.UniversalClient) (http.Handler, error) {
	systemRepo := repository.NewSystem(l.systemCli)

	authenticationCache := repository.NewAuthenticationCache(time.Duration(config.Caching.AuthenticationDataInSec) * time.Second)
	authentication := service.NewAuthentication(authenticationCache, systemRepo)

	adminService := service.NewAdmin(config.Secrets.AdminTokenSecret)

	authorizationCache := repository.NewAuthorizationCache(time.Duration(config.Caching.AuthorizationDataInSec) * time.Second)
	authorization := service.NewAuthorization(authorizationCache, systemRepo)

	dailyLimitRepo := repository.NewDailyLimit(redisCli)
	dailyLimitService := service.NewDailyLimit(dailyLimitRepo, config.DailyLimits)

	throttlingRepo := repository.NewThrottling(redisCli)
	throttlingService := service.NewThrottling(throttlingRepo, config.Throttling)

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
		default:
			return nil, errors.Errorf("not supported protocol %s", location.Protocol)
		}

		handler := middleware.Chain(
			proxyFunc,
			middleware.RequestId(),
			middleware.Logger(l.logger, config.Logging.RequestLogEnable, config.Logging.BodyLogEnable),
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
				middleware.Logger(l.logger, config.Logging.RequestLogEnable, config.Logging.BodyLogEnable),
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
