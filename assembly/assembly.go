package assembly

import (
	"context"

	"isp-gate-service/conf"
	"isp-gate-service/routes"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/app"
	"github.com/txix-open/isp-kit/bootstrap"
	"github.com/txix-open/isp-kit/cluster"
	"github.com/txix-open/isp-kit/grpc/client"
	"github.com/txix-open/isp-kit/http"
	"github.com/txix-open/isp-kit/lb"
	"github.com/txix-open/isp-kit/log"
)

type Assembly struct {
	boot      *bootstrap.Bootstrap
	server    *http.Server
	logger    *log.Adapter
	routes    *routes.Routes
	systemCli *client.Client
	adminCli  *client.Client
	lockerCli *client.Client

	locations                   []conf.Location
	grpcClientByModuleName      map[string]*client.Client
	httpHostManagerByModuleName map[string]*lb.RoundRobin
}

func New(boot *bootstrap.Bootstrap) (*Assembly, error) {
	server := http.NewServer(boot.App.Logger())

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
	lockerCli, err := client.Default()
	if err != nil {
		return nil, errors.WithMessage(err, "create isp-lock-service client")
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
		lockerCli:                   lockerCli,
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
	a.logger.SetLevel(newCfg.Logging.LogLevel)

	locator := NewLocator(
		a.logger,
		a.grpcClientByModuleName,
		a.httpHostManagerByModuleName,
		a.routes,
		a.systemCli,
		a.adminCli,
		a.lockerCli,
	)
	handler, err := locator.Handler(newCfg, a.locations)
	if err != nil {
		return errors.WithMessage(err, "locator handler")
	}

	a.server.Upgrade(handler)

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
	eventHandler.RequireModule("isp-lock-service", a.lockerCli)

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
	closers = append(closers, a.systemCli, a.adminCli, a.lockerCli)

	return closers
}
