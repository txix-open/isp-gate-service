package main

import (
	"context"
	"os"

	"github.com/integration-system/go-cmp/cmp"
	"github.com/integration-system/isp-lib/v2/bootstrap"
	"github.com/integration-system/isp-lib/v2/config"
	"github.com/integration-system/isp-lib/v2/config/schema"
	"github.com/integration-system/isp-lib/v2/metric"
	"github.com/integration-system/isp-lib/v2/structure"
	"github.com/integration-system/isp-lib/v2/utils"
	logrus "github.com/integration-system/isp-log"
	"github.com/integration-system/isp-log/stdcodes"
	"github.com/pkg/errors"
	"isp-gate-service/accounting"
	"isp-gate-service/authenticate"
	"isp-gate-service/conf"
	"isp-gate-service/log"
	"isp-gate-service/proxy"
	"isp-gate-service/redis"
	"isp-gate-service/repository"
	"isp-gate-service/routing"
	"isp-gate-service/server"
	"isp-gate-service/service"
	"isp-gate-service/service/matcher"
)

var (
	version = "0.1.0"

	logger *log.Adapter
)

func main() {
	cfg := config.InitConfig(&conf.Configuration{}).(*conf.Configuration)

	bs := bootstrap.
		ServiceBootstrap(&conf.Configuration{}, &conf.RemoteConfig{}).
		OnLocalConfigLoad(onLocalConfigLoad).
		DefaultRemoteConfigPath(schema.ResolveDefaultConfigPath("default_remote_config.json")).
		DeclareMe(makeDeclaration).
		SocketConfiguration(socketConfiguration).
		RequireRoutes(handleRouteUpdate).
		SubscribeBroadcastEvent(bootstrap.ListenRestartEvent())

	requiredModules, err := proxy.InitProxies(cfg.Locations)
	if err != nil {
		logrus.Fatal(stdcodes.ModuleInvalidLocalConfig, err)
	}
	for module, consumer := range requiredModules {
		bs.RequireModule(module, consumer, false)
	}
	bs.RequireModule(cfg.ModuleName, accounting.NewConnectionConsumer, false).
		OnShutdown(onShutdown).
		OnRemoteConfigReceive(onRemoteConfigReceive).
		Run()
}

func onLocalConfigLoad(_ *conf.Configuration) {

}

func onRemoteConfigReceive(remoteConfig, oldRemoteConfig *conf.RemoteConfig) {
	repository.DbClient.ReceiveConfiguration(remoteConfig.Database)
	matcher.JournalMethods = matcher.NewAtLeastOneMatcher(remoteConfig.JournalSetting.MethodsPatterns)

	redis.Client.ReceiveConfiguration(remoteConfig.Redis)
	authenticate.ReceiveConfiguration(remoteConfig.AuthCacheSetting)
	accounting.ReceiveConfiguration(remoteConfig.AccountingSetting)

	metric.InitCollectors(remoteConfig.Metrics, oldRemoteConfig.Metrics)
	metric.InitHttpServer(remoteConfig.Metrics)
	service.Metrics.Init()

	oldLogger := setLogger(remoteConfig.JournalSetting.Journal, oldRemoteConfig.JournalSetting.Journal)
	defer func() {
		if oldLogger != nil {
			_ = oldLogger.Close()
		}
	}()

	isDifferentSettingForHttpServ :=
		!cmp.Equal(remoteConfig.HttpSetting, oldRemoteConfig.HttpSetting) || oldLogger != nil
	server.Http.Init(isDifferentSettingForHttpServ, remoteConfig.HttpSetting.GetMaxRequestBodySize(), logger)
}

func socketConfiguration(cfg interface{}) structure.SocketConfiguration {
	appConfig := cfg.(*conf.Configuration)
	return structure.SocketConfiguration{
		Host:   appConfig.ConfigServiceAddress.IP,
		Port:   appConfig.ConfigServiceAddress.Port,
		Secure: false,
		UrlParams: map[string]string{
			"module_name": appConfig.ModuleName,
		},
	}
}

func onShutdown(_ context.Context, _ os.Signal) {
	server.Http.Close()
	accounting.Close()
	proxy.Close()
	_ = redis.Client.Close()
	if logger != nil {
		_ = logger.Close()
	}
}

func handleRouteUpdate(configs structure.RoutingConfig) bool {
	routing.InitRoutes(configs)
	return true
}

func makeDeclaration(localConfig interface{}) bootstrap.ModuleInfo {
	cfg := localConfig.(*conf.Configuration)
	return bootstrap.ModuleInfo{
		ModuleName:       cfg.ModuleName,
		ModuleVersion:    version,
		GrpcOuterAddress: cfg.HttpOuterAddress,
		Endpoints:        []structure.EndpointDescriptor{},
	}
}

func setLogger(loggerCfg, oldLoggerCfg conf.JorunalConfig) *log.Adapter {
	if cmp.Equal(loggerCfg, oldLoggerCfg) {
		return nil
	}

	var err error
	oldLogger := logger
	if utils.DEV {
		logger, err = log.New()
	} else {
		logger, err = log.New(log.WithFileRotation(log.Rotation{
			File:      loggerCfg.Filename,
			MaxSizeMb: loggerCfg.MaxSizeMb,
			Compress:  loggerCfg.Compress,
		}))
	}
	if err != nil {
		logrus.Fatal(stdcodes.ModuleRunFatalError, errors.WithMessage(err, "set logger"))
	}

	return oldLogger
}
