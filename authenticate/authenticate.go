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

func Do(ctx *fasthttp.RequestCtx) (int32, error) {
	path := ctx.Path()
	pathStr := getPathWithoutPrefix(path)

	if _, ok := routing.AllMethods[pathStr]; !ok {
		return 0, createError(codes.Unimplemented)
	}

	for _, notExpectedHeader := range notExpectedHeaders {
		ctx.Request.Header.Del(notExpectedHeader)
	}

	appToken := ctx.Request.Header.Peek(utils.ApplicationTokenHeader)

	var (
		err   error
		appId int32 = -1
	)
	if len(appToken) == 0 {
		return 0, createError(codes.Unauthenticated)
	} else if config.GetRemote().(*conf.RemoteConfig).Secrets.VerifyAppToken {
		if appId, err = verifyToken.Application(string(appToken)); err != nil || appId == 0 {
			return 0, createError(codes.Unauthenticated, "invalid token")
		}
	}

	verifiableKeys, err := auth.verify.ApplicationToken(string(appToken))
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError(codes.Internal)
	}
	if len(verifiableKeys) != 4 {
		return 0, createError(codes.Unauthenticated, "unknown token")
	}

	applicationId, err := strconv.Atoi(verifiableKeys[utils.ApplicationIdHeader])
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, errors.WithMessagef(err, "parse appId from redis"))
		return 0, createError(codes.Internal)
	}
	if appId != -1 && int32(applicationId) != appId {
		return 0, createError(codes.Unauthenticated, "unknown application identity")
	}

	verifiableKeys[utils.DeviceTokenHeader] = string(ctx.Request.Header.Peek(utils.DeviceTokenHeader))
	verifiableKeys[utils.UserTokenHeader] = string(ctx.Request.Header.Peek(utils.UserTokenHeader))

	permittedToCall := false
	verifiableKeys, permittedToCall, err = auth.verify.Identity(verifiableKeys, pathStr)
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError(codes.Internal)
	}
	if permittedToCall {
		return 0, createError(codes.PermissionDenied)
	}

	for key, value := range verifiableKeys {
		if value != "" {
			ctx.Request.Header.Set(key, value)
		}
	}

	if _, ok := routing.InnerMethods[pathStr]; ok {
		adminToken := ctx.Request.Header.Peek("x-auth-admin") //todo const key
		if verifyToken.Admin(string(adminToken)) != nil {
			return 0, createError(codes.PermissionDenied)
		}
	}

	return appId, nil
}

func getPathWithoutPrefix(path []byte) string {
	firstFound := false
	for i, value := range path {
		if value == '/' {
			if firstFound {
				return string(path[i+1:])
			} else {
				firstFound = true
			}
		}
	}
	return ""
}
