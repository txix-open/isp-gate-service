package main

import (
	"github.com/txix-open/isp-kit/bootstrap"
	"github.com/txix-open/isp-kit/shutdown"
	"isp-gate-service/assembly"
	"isp-gate-service/conf"
)

var (
	version = "1.0.0"
)

func main() {
	boot := bootstrap.New(version, conf.Remote{}, nil)
	app := boot.App
	logger := app.Logger()

	assembly, err := assembly.New(boot)
	if err != nil {
		logger.Fatal(app.Context(), err)
	}
	app.AddRunners(assembly.Runners()...)
	app.AddClosers(assembly.Closers()...)

	shutdown.On(func() {
		logger.Info(app.Context(), "starting shutdown")
		app.Shutdown()
		logger.Info(app.Context(), "shutdown completed")
	})

	err = app.Run()
	if err != nil {
		app.Shutdown()
		logger.Fatal(app.Context(), err)
	}
}
