package main

import (
	"context"
	"log"
	"os"

	"github.com/integration-system/isp-lib/v2/bootstrap"
	"github.com/integration-system/isp-lib/v2/config"
	"github.com/integration-system/isp-lib/v2/config/schema"
	"github.com/integration-system/isp-lib/v2/metric"
	"github.com/integration-system/isp-lib/v2/structure"
	"github.com/integration-system/isp-log/stdcodes"
	"isp-gate-service/accounting"
	"isp-gate-service/authenticate"
	"isp-gate-service/conf"
	"isp-gate-service/invoker"
	"isp-gate-service/model"
	"isp-gate-service/proxy"
	"isp-gate-service/redis"
	"isp-gate-service/routing"
	"isp-gate-service/server"
	"isp-gate-service/service"
	"isp-gate-service/service/matcher"
)

var (
	version = "0.1.0"
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
		RequireModule("journal", invoker.Journal.ReceiveServiceAddressList, false).
		SubscribeBroadcastEvent(bootstrap.ListenRestartEvent())

	requiredModules, err := proxy.InitProxies(cfg.Locations)
	if err != nil {
		log.Fatal(stdcodes.ModuleInvalidLocalConfig, err)
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
	localCfg := config.Get().(*conf.Configuration)

	model.DbClient.ReceiveConfiguration(remoteConfig.Database)
	invoker.Journal.ReceiveConfiguration(remoteConfig.JournalSetting.Journal, localCfg.ModuleName)
	matcher.JournalMethods = matcher.NewAtLeastOneMatcher(remoteConfig.JournalSetting.MethodsPatterns)

	redis.Client.ReceiveConfiguration(remoteConfig.Redis)
	authenticate.ReceiveConfiguration(remoteConfig.AuthCacheSetting)
	accounting.ReceiveConfiguration(remoteConfig.AccountingSetting)

	metric.InitCollectors(remoteConfig.Metrics, oldRemoteConfig.Metrics)
	metric.InitHttpServer(remoteConfig.Metrics)
	service.Metrics.Init()

	server.Http.Init(remoteConfig.HttpSetting, oldRemoteConfig.HttpSetting)
}

func socketConfiguration(cfg interface{}) structure.SocketConfiguration {
	appConfig := cfg.(*conf.Configuration)
	return structure.SocketConfiguration{
		Host:   appConfig.ConfigServiceAddress.IP,
		Port:   appConfig.ConfigServiceAddress.Port,
		Secure: false,
		UrlParams: map[string]string{
			"module_name":   appConfig.ModuleName,
			"instance_uuid": appConfig.InstanceUuid,
		},
	}
}

func onShutdown(_ context.Context, _ os.Signal) {
	server.Http.Close()
	accounting.Close()
	proxy.Close()
	_ = redis.Client.Close()
	_ = invoker.Journal.Close()
}

func handleRouteUpdate(configs structure.RoutingConfig) bool {
	routing.InitRoutes(configs)
	err := proxy.InitProxiesFromConfigs(configs)
	if err != nil {
		log.Fatal(err)
	}
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
