package main

import (
	"context"
	"github.com/valyala/fasthttp"
	"sync"
	"time"

	"github.com/integration-system/isp-lib/bootstrap"
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/config/schema"
	"github.com/integration-system/isp-lib/metric"
	"github.com/integration-system/isp-lib/structure"
	log "github.com/integration-system/isp-log"

	"isp-gate-service/conf"
	"isp-gate-service/journal"
	"isp-gate-service/log_code"
	"isp-gate-service/proxy"
	"isp-gate-service/service"
	"os"
)

var (
	version = "0.1.0"
	date    = "undefined"

	srvLock = sync.Mutex{}
	httpSrv *fasthttp.Server
)

func main() {
	cfg := config.InitConfig(&conf.Configuration{}).(*conf.Configuration)

	bs := bootstrap.
		ServiceBootstrap(&conf.Configuration{}, &conf.RemoteConfig{}).
		OnLocalConfigLoad(onLocalConfigLoad).
		DefaultRemoteConfigPath(schema.ResolveDefaultConfigPath("default_remote_config.json")).
		DeclareMe(makeDeclaration).
		SocketConfiguration(socketConfiguration).
		RequireModule(journal.RequiredModule())

	for _, location := range cfg.Locations {
		if p, err := proxy.Init(location); err != nil {
			log.Fatal(log_code.ErrorLocalConfig, err)
		} else {
			bs.RequireModule(location.TargetModule, p.Consumer, true)
		}
	}

	bs.OnShutdown(onShutdown).
		OnRemoteConfigReceive(onRemoteConfigReceive).
		Run()
}

func onLocalConfigLoad(cfg *conf.Configuration) {
	log.Infof(log_code.InfoOnLocalConfigLoad, "Outer http address is %s", cfg.HttpOuterAddress.GetAddress())
}

func onRemoteConfigReceive(remoteConfig, oldRemoteConfig *conf.RemoteConfig) {
	localCfg := config.Get().(*conf.Configuration)

	journal.Client.ReceiveConfiguration(remoteConfig.Journal, localCfg.ModuleName)

	service.JournalMethodsMatcher = service.NewCacheableMethodMatcher(remoteConfig.JournalingMethodsPatterns)

	createServer(remoteConfig)
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
	for _, p := range proxy.ProxyStore {
		p.Close()
	}
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

func createServer(remoteConfig *conf.RemoteConfig) {
	srvLock.Lock()
	if httpSrv != nil {
		if err := httpSrv.Shutdown(); err != nil {
			log.Warn(log_code.WarnCreateRestServerHttpSrvShutdown, err)
		}
	}
	maxRequestBodySize := remoteConfig.GetMaxRequestBodySize()
	localConfig := config.Get().(*conf.Configuration)
	restAddress := localConfig.HttpInnerAddress.GetAddress()
	httpSrv = &fasthttp.Server{
		Handler:            proxy.Handle,
		WriteTimeout:       time.Second * 60,
		ReadTimeout:        time.Second * 60,
		MaxRequestBodySize: int(maxRequestBodySize),
	}
	go func() {
		if err := httpSrv.ListenAndServe(restAddress); err != nil {
			log.Error(log_code.ErrorCreateRestServerHttpSrvListenAndServe, err)
		}
	}()
	srvLock.Unlock()
}
