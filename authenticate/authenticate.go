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

const (
	// TODO move to isp-lib
	AdminTokenHeader = "x-auth-admin"
)

var notExpectedHeaders = []string{
	utils.SystemIdHeader,
	utils.DomainIdHeader,
	utils.ServiceIdHeader,
	utils.ApplicationIdHeader,
	utils.UserIdHeader,
	utils.DeviceIdHeader,
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

	appToken := getParam(utils.ApplicationTokenHeader, &ctx.Request)

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

	verifiableKeys[utils.DeviceTokenHeader] = string(getParam(utils.DeviceTokenHeader, &ctx.Request))
	verifiableKeys[utils.UserTokenHeader] = string(getParam(utils.UserTokenHeader, &ctx.Request))

	if verifiableKeys[utils.UserTokenHeader] != "" {
		userId, err := verifyToken.User(verifiableKeys[utils.UserTokenHeader])
		if err != nil {
			return 0, createError("forbidden", codes.PermissionDenied)
		}
		verifiableKeys[utils.UserIdHeader] = userId
	}

	permittedToCall := false
	validUserId := true
	verifiableKeys, permittedToCall, validUserId, err = auth.verify.Identity(verifiableKeys, path)
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError("internal server error", codes.Internal)
	}
	if !validUserId {
		msg := "doesn't match user id"
		log.Error(log_code.ErrorAuthenticate, msg)
		return 0, createError(msg, codes.PermissionDenied)
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
		adminToken := getParam(AdminTokenHeader, &ctx.Request)
		if verifyToken.Admin(string(adminToken)) != nil {
			return 0, createError("forbidden", codes.PermissionDenied)
		}
	}
	if verifiableKeys[utils.UserIdHeader] == "" {
		if _, ok := routing.AuthUserMethods[path]; ok {
			return 0, createError("forbidden", codes.PermissionDenied)
		}
	}

	//TODO backward capability for isp-config-service 1.x.x
	uuid := config.Get().(*conf.Configuration).InstanceUuid
	ctx.Request.Header.Set(utils.InstanceIdHeader, uuid)

	return appId, nil
}

func getParam(key string, req *fasthttp.Request) []byte {
	val := req.Header.Peek(key)
	if len(val) != 0 {
		return val
	}
	return req.URI().QueryArgs().Peek(key)
}
