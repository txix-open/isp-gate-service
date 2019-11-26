package main

import (
	"context"
	"github.com/integration-system/isp-lib/bootstrap"
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/config/schema"
	"github.com/integration-system/isp-lib/metric"
	"github.com/integration-system/isp-lib/structure"
	log "github.com/integration-system/isp-log"
	"github.com/integration-system/isp-log/stdcodes"
	"isp-gate-service/accounting"
	"isp-gate-service/authenticate"
	"isp-gate-service/conf"
	"isp-gate-service/journal"
	"isp-gate-service/model"
	"isp-gate-service/proxy"
	"isp-gate-service/redis"
	"isp-gate-service/routing"
	"isp-gate-service/server"
	"isp-gate-service/service"
	"isp-gate-service/service/matcher"
	"os"
)

var (
	version = "0.1.0"
	date    = "undefined"
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
		RequireModule(journal.RequiredModule())

	requiredModules := getRequiredModulesByLocations(cfg.Locations)
	for module, consumer := range requiredModules {
		bs.RequireModule(module, consumer, false)
	}

	bs.RequireModule(cfg.ModuleName, accounting.Consumer, false).
		OnShutdown(onShutdown).
		OnRemoteConfigReceive(onRemoteConfigReceive).
		Run()
}

func onLocalConfigLoad(cfg *conf.Configuration) {

}

func onRemoteConfigReceive(remoteConfig, oldRemoteConfig *conf.RemoteConfig) {
	localCfg := config.Get().(*conf.Configuration)

	model.DbClient.ReceiveConfiguration(remoteConfig.Database)
	journal.Client.ReceiveConfiguration(remoteConfig.JournalSetting.Journal, localCfg.ModuleName)
	matcher.JournalMethods = matcher.NewAtLeastOneMatcher(remoteConfig.JournalSetting.MethodsPatterns)

	redis.Client.ReceiveConfiguration(remoteConfig.Redis)
	authenticate.ReceiveConfiguration(remoteConfig.AuthCacheSetting)
	accounting.ReceiveConfiguration(remoteConfig.AccountingSetting)

	metric.InitCollectors(remoteConfig.Metrics, oldRemoteConfig.Metrics)
	metric.InitHttpServer(remoteConfig.Metrics)
	service.Metrics.Init()

	server.Http.Init(remoteConfig.ServerSetting.GetMaxRequestBodySize())
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
	redis.Client.Close()
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
		Handlers:         []interface{}{},
	}
}

func getRequiredModulesByLocations(locations []conf.Location) map[string]func([]structure.AddressConfiguration) bool {
	locationsByTargetModule := conf.GetLocationsByTargetModule(locations)
	requiredModules := make(map[string]func([]structure.AddressConfiguration) bool)

	for targetModule, locations := range locationsByTargetModule {
		consumerStorage := make([]func([]structure.AddressConfiguration) bool, len(locations))
		for i, location := range locations {
			if p, err := proxy.Init(location.Protocol, location.PathPrefix, location.SkipAuth); err != nil {
				log.Fatal(stdcodes.ModuleInvalidLocalConfig, err)
			} else {
				consumerStorage[i] = p.Consumer
			}
		}
		requiredModules[targetModule] = aggregateConsumers(consumerStorage...)
	}

	return requiredModules
}

func aggregateConsumers(consumers ...func([]structure.AddressConfiguration) bool) func([]structure.AddressConfiguration) bool {
	return func(list []structure.AddressConfiguration) bool {
		for _, consumer := range consumers {
			consumer(list)
		}
		return true
	}
}
