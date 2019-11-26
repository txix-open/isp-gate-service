package authenticate

import (
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/utils"
	log "github.com/integration-system/isp-log"
	"github.com/integration-system/isp-log/stdcodes"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/conf"
	"isp-gate-service/log_code"
	"isp-gate-service/routing"
	"isp-gate-service/service/veritification"
	"strconv"
	"time"
)

var notExpectedHeaders = []string{
	utils.SystemIdHeader, utils.DomainIdHeader, utils.ServiceIdHeader, utils.ApplicationIdHeader,
}

var auth = authenticate{}

type authenticate struct {
	verify veritification.Verify
}

func ReceiveConfiguration(conf conf.Cache) {
	if conf.EnableCash {
		if timeout, err := time.ParseDuration(conf.EvictTimeout); err != nil {
			log.Fatalf(stdcodes.ModuleInvalidRemoteConfig, "invalid timeout '%s'", timeout)
		} else {
			auth.verify = veritification.NewCacheableVerify(timeout)
		}
	} else {
		auth.verify = veritification.NewRuntimeVerify()
	}
}

func Do(ctx *fasthttp.RequestCtx, path string) (int32, error) {
	for _, notExpectedHeader := range notExpectedHeaders {
		ctx.Request.Header.Del(notExpectedHeader)
	}

	appToken := ctx.Request.Header.Peek(utils.ApplicationTokenHeader)

	var (
		err   error
		appId int32 = -1
	)

	if len(appToken) == 0 {
		return 0, createError("unauthorized", codes.Unauthenticated)
	} else if config.GetRemote().(*conf.RemoteConfig).TokensSetting.ApplicationVerify {
		if appId, err = verifyToken.Application(string(appToken)); err != nil || appId == 0 {
			return 0, createError("unauthorized", codes.Unauthenticated, "invalid token")
		}
	}

	verifiableKeys, err := auth.verify.ApplicationToken(string(appToken))
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError("internal Server error", codes.Internal)
	}
	if len(verifiableKeys) != 4 {
		return 0, createError("unauthorized", codes.Unauthenticated, "unknown token")
	}

	applicationId, err := strconv.Atoi(verifiableKeys[utils.ApplicationIdHeader])
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, errors.WithMessagef(err, "parse appId from redis"))
		return 0, createError("internal Server error", codes.Internal)
	}
	if appId != -1 && int32(applicationId) != appId {
		return 0, createError("unauthorized", codes.Unauthenticated, "unknown application identity")
	}

	verifiableKeys[utils.DeviceTokenHeader] = string(ctx.Request.Header.Peek(utils.DeviceTokenHeader))
	verifiableKeys[utils.UserTokenHeader] = string(ctx.Request.Header.Peek(utils.UserTokenHeader))

	permittedToCall := false
	verifiableKeys, permittedToCall, err = auth.verify.Identity(verifiableKeys, path)
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError("internal Server error", codes.Internal)
	}
	if permittedToCall {
		return 0, createError("forbidden", codes.PermissionDenied)
	}

	for key, value := range verifiableKeys {
		if value != "" {
			ctx.Request.Header.Set(key, value)
		}
	}

	if _, ok := routing.InnerMethods[path]; ok {
		adminToken := ctx.Request.Header.Peek("x-auth-admin") //todo const key
		if verifyToken.Admin(string(adminToken)) != nil {
			return 0, createError("forbidden", codes.PermissionDenied)
		}
	}

	//TODO backward capability for isp-config-service 1.x.x
	uuid := config.Get().(*conf.Configuration).InstanceUuid
	ctx.Request.Header.Set(utils.InstanceIdHeader, uuid)

	return appId, nil
}
