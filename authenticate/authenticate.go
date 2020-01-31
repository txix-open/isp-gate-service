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
		return 0, createError("unauthorized", codes.Unauthenticated, "application token required")
	} else if config.GetRemote().(*conf.RemoteConfig).TokensSetting.ApplicationVerify {
		if appId, err = verifyToken.Application(string(appToken)); err != nil || appId == 0 {
			return 0, createError("unauthorized", codes.Unauthenticated, "received invalid application token")
		}
	}

	verifiableKeys, err := auth.verify.ApplicationToken(string(appToken))
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError("internal Server error", codes.Internal)
	}
	if len(verifiableKeys) != 4 {
		return 0, createError("unauthorized", codes.Unauthenticated, "received unexpected identities")
	}

	applicationId, err := strconv.Atoi(verifiableKeys[utils.ApplicationIdHeader])
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, errors.WithMessagef(err, "parse appId from redis"))
		return 0, createError("internal Server error", codes.Internal)
	}
	if appId != -1 && int32(applicationId) != appId {
		return 0, createError("unauthorized", codes.Unauthenticated, "received unexpected application identity")
	}

	userToken := string(getParam(utils.UserTokenHeader, &ctx.Request))
	if userToken != "" {
		userId, err := verifyToken.User(userToken)
		if err != nil {
			return 0, createError("unauthorized", codes.Unauthenticated, "received invalid user token")
		}
		verifiableKeys[utils.UserIdHeader] = userId
	}

	verifiableKeys[utils.UserTokenHeader] = userToken
	verifiableKeys[utils.DeviceTokenHeader] = string(getParam(utils.DeviceTokenHeader, &ctx.Request))
	verifiableKeys, err = auth.verify.Identity(verifiableKeys, path)
	if err != nil {
		switch err {
		case veritification.ErrorInvalidUserId:
			return 0, createError("unauthorized", codes.Unauthenticated, err.Error())
		case veritification.ErrorPermittedToCallUser, veritification.ErrorPermittedToCallApplication:
			return 0, createError("forbidden", codes.PermissionDenied, err.Error())
		default:
			return 0, createError("internal server error", codes.Internal)
		}
	}

	for key, value := range verifiableKeys {
		if value != "" {
			ctx.Request.Header.Set(key, value)
		}
	}

	if _, ok := routing.InnerMethods[path]; ok {
		adminToken := string(getParam(AdminTokenHeader, &ctx.Request))
		if adminToken == "" {
			return 0, createError("unauthorized", codes.Unauthenticated, "admin token required")
		}
		if verifyToken.Admin(adminToken) != nil {
			return 0, createError("unauthorized", codes.Unauthenticated, "received invalid admin token")
		}
	}
	if verifiableKeys[utils.UserIdHeader] == "" {
		if _, ok := routing.AuthUserMethods[path]; ok {
			return 0, createError("unauthorized", codes.Unauthenticated, "user token required")
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
