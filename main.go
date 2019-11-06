package main

import (
	"context"
	"github.com/integration-system/isp-lib/bootstrap"
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/config/schema"
	"github.com/integration-system/isp-lib/metric"
	"github.com/integration-system/isp-lib/structure"
	log "github.com/integration-system/isp-log"
	"isp-gate-service/approve"
	"isp-gate-service/conf"
	"isp-gate-service/journal"
	"isp-gate-service/log_code"
	"isp-gate-service/proxy"
	"isp-gate-service/redis"
	"isp-gate-service/routing"
	"isp-gate-service/server"
	"isp-gate-service/service"
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

	for _, location := range cfg.Locations {
		if p, err := proxy.Init(location); err != nil {
			log.Fatal(log_code.FatalLocalConfig, err)
		} else {
			bs.RequireModule(location.TargetModule, p.Consumer, false)
		}
	}

	bs.OnShutdown(onShutdown).
		OnRemoteConfigReceive(onRemoteConfigReceive).
		Run()
}

func onLocalConfigLoad(cfg *conf.Configuration) {

}

func onRemoteConfigReceive(remoteConfig, oldRemoteConfig *conf.RemoteConfig) {
	localCfg := config.Get().(*conf.Configuration)

	journal.Client.ReceiveConfiguration(remoteConfig.Journal, localCfg.ModuleName)
	redis.Client.ReceiveConfiguration(remoteConfig.Redis)
	server.Http.Init(remoteConfig.ServerSetting.MaxRequestBodySizeBytes)
	approve.ReceiveConfiguration(remoteConfig.ApproveSetting)

	service.JournalMethodsMatcher = service.NewCacheableMethodMatcher(remoteConfig.JournalingMethodsPatterns)

	metric.InitCollectors(remoteConfig.Metrics, oldRemoteConfig.Metrics)
	metric.InitHttpServer(remoteConfig.Metrics)
	//metric.InitStatusChecker("router-grpc", helper.GetRoutersAndStatus)
	service.Metrics.Init()
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
	proxy.Close()
	redis.Client.Close()
	server.Http.Close()
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
