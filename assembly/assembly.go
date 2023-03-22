package assembly

import (
	"context"

	"github.com/redis/go-redis/v9"
	"isp-gate-service/conf"
	"isp-gate-service/routes"

	"github.com/integration-system/isp-kit/app"
	"github.com/integration-system/isp-kit/bootstrap"
	"github.com/integration-system/isp-kit/cluster"
	"github.com/integration-system/isp-kit/grpc/client"
	"github.com/integration-system/isp-kit/http"
	"github.com/integration-system/isp-kit/lb"
	"github.com/integration-system/isp-kit/log"
	"github.com/pkg/errors"
)

type Assembly struct {
	boot      *bootstrap.Bootstrap
	server    *http.Server
	logger    *log.Adapter
	routes    *routes.Routes
	redisCli  redis.UniversalClient
	systemCli *client.Client
	adminCli  *client.Client

	locations                   []conf.Location
	grpcClientByModuleName      map[string]*client.Client
	httpHostManagerByModuleName map[string]*lb.RoundRobin
}

func New(boot *bootstrap.Bootstrap) (*Assembly, error) {
	server := http.NewServer()

	localConfig := conf.Local{}
	err := boot.App.Config().Read(&localConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "read local config")
	}

	grpcClientByModuleName := make(map[string]*client.Client)
	httpHostManagerByModuleName := make(map[string]*lb.RoundRobin)
	for _, location := range localConfig.Locations {
		switch location.Protocol {
		case conf.GrpcProtocol:
			cli, err := client.Default()
			if err != nil {
				return nil, errors.WithMessage(err, "new grpc client")
			}
			grpcClientByModuleName[location.TargetModule] = cli
		case conf.HttpProtocol, conf.WsProtocol:
			rb := lb.NewRoundRobin(nil)
			httpHostManagerByModuleName[location.TargetModule] = rb
		default:
			return nil, errors.Errorf("unexpected location protocol: %s", location.Protocol)
		}
	}

	systemCli, err := client.Default()
	if err != nil {
		return nil, errors.WithMessage(err, "create system cli")
	}

	adminCli, err := client.Default()
	if err != nil {
		return nil, errors.WithMessage(err, "create admin cli")
	}

	return &Assembly{
		boot:                        boot,
		server:                      server,
		logger:                      boot.App.Logger(),
		routes:                      routes.NewRoutes(),
		locations:                   localConfig.Locations,
		grpcClientByModuleName:      grpcClientByModuleName,
		httpHostManagerByModuleName: httpHostManagerByModuleName,
		systemCli:                   systemCli,
		adminCli:                    adminCli,
	}, nil
}

func (a *Assembly) ReceiveConfig(ctx context.Context, remoteConfig []byte) error {
	var (
		newCfg  conf.Remote
		prevCfg conf.Remote
	)
	err := a.boot.RemoteConfig.Upgrade(remoteConfig, &newCfg, &prevCfg)
	if err != nil {
		a.logger.Fatal(ctx, errors.WithMessage(err, "upgrade remote config"))
	}
	err = newCfg.Validate()
	if err != nil {
		a.logger.Fatal(ctx, errors.WithMessage(err, "invalid remote config"))
	}

	a.logger.SetLevel(newCfg.Logging.LogLevel)

	var newRedisCli redis.UniversalClient
	if newCfg.Redis != nil {
		newRedisCli = a.redisClient(*newCfg.Redis)
	}

	locator := NewLocator(a.logger, a.grpcClientByModuleName, a.httpHostManagerByModuleName, a.routes, a.systemCli, a.adminCli)

	handler, err := locator.Handler(newCfg, a.locations, newRedisCli)
	if err != nil {
		return errors.WithMessage(err, "locator handler")
	}

	a.server.Upgrade(handler)

	if a.redisCli != nil {
		_ = a.redisCli.Close()
		a.redisCli = newRedisCli
	}

	return nil
}

func (a *Assembly) Runners() []app.Runner {
	eventHandler := cluster.NewEventHandler().
		RoutesReceiver(a.routes).
		RemoteConfigReceiver(a)

	for moduleName, upgrader := range a.grpcClientByModuleName {
		eventHandler.RequireModule(moduleName, upgrader)
	}
	for moduleName, upgrader := range a.httpHostManagerByModuleName {
		eventHandler.RequireModule(moduleName, upgrader)
	}

	eventHandler.RequireModule("isp-system-service", a.systemCli)
	eventHandler.RequireModule("msp-admin-service", a.adminCli)

	return []app.Runner{
		app.RunnerFunc(func(ctx context.Context) error {
			return a.server.ListenAndServe(a.boot.BindingAddress)
		}),
		app.RunnerFunc(func(ctx context.Context) error {
			return a.boot.ClusterCli.Run(ctx, eventHandler)
		}),
	}
}

func (a *Assembly) Closers() []app.Closer {
	closers := []app.Closer{
		a.boot.ClusterCli,
		app.CloserFunc(func() error {
			return a.server.Shutdown(context.Background())
		}),
	}
	for _, cliCloser := range a.grpcClientByModuleName {
		closers = append(closers, cliCloser)
	}
	closers = append(closers, a.systemCli, a.adminCli, app.CloserFunc(func() error {
		if a.redisCli != nil {
			return a.redisCli.Close()
		}
		return nil
	}))

	return closers
}

func (a *Assembly) redisClient(config conf.Redis) redis.UniversalClient {
	if config.Sentinel != nil {
		return redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       config.Sentinel.MasterName,
			SentinelAddrs:    config.Sentinel.Addresses,
			SentinelUsername: config.Sentinel.Username,
			SentinelPassword: config.Sentinel.Password,
			Username:         config.Username,
			Password:         config.Password,
		})
	}
	return redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Username: config.Username,
		Password: config.Password,
	})
}
