package assembly

import (
	"context"

	"github.com/integration-system/isp-kit/app"
	"github.com/integration-system/isp-kit/bootstrap"
	"github.com/integration-system/isp-kit/cluster"
	"github.com/integration-system/isp-kit/grpc/client"
	"github.com/integration-system/isp-kit/http"
	"github.com/integration-system/isp-kit/lb"
	"github.com/integration-system/isp-kit/log"
	"github.com/pkg/errors"
	"isp-gate-service/conf"
	"isp-gate-service/routes"
)

const defaultHost = "127.0.0.1"

type Assembly struct {
	boot   *bootstrap.Bootstrap
	server *http.Server
	logger *log.Adapter
	routes *routes.Routes

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
		case conf.HttpProtocol:
			rb := lb.NewRoundRobin([]string{defaultHost})
			httpHostManagerByModuleName[location.TargetModule] = rb
		case conf.WebsocketProtocol:
		default:
			return nil, errors.Errorf("unexpected location protocol: %s", location.Protocol)
		}
	}

	return &Assembly{
		boot:                        boot,
		server:                      server,
		logger:                      boot.App.Logger(),
		routes:                      routes.NewRoutes(),
		locations:                   localConfig.Locations,
		grpcClientByModuleName:      grpcClientByModuleName,
		httpHostManagerByModuleName: httpHostManagerByModuleName,
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

	a.logger.SetLevel(newCfg.LogLevel)

	locator := NewLocator(a.logger, a.grpcClientByModuleName, a.httpHostManagerByModuleName, a.routes)

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
			return a.server.Shutdown(a.boot.App.Context())
		}),
	}
	for _, cliCloser := range a.grpcClientByModuleName {
		closers = append(closers, cliCloser)
	}

	return closers
}
