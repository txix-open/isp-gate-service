package authenticate

import (
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/utils"
	log "github.com/integration-system/isp-log"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/conf"
	"isp-gate-service/log_code"
	"isp-gate-service/routing"
	"isp-gate-service/service/veritification"
	"strconv"
	"strings"
	"time"
)

var notExpectedHeaders = []string{
	utils.SystemIdHeader, utils.DomainIdHeader, utils.ServiceIdHeader, utils.ApplicationIdHeader,
}

var auth = authenticate{}

type authenticate struct {
	verify veritification.Verify
}

func ReceiveConfiguration(conf conf.Authenticate) {
	if conf.EnableCash {
		if timeout, err := time.ParseDuration(conf.Timeout); err != nil {
			log.Fatalf(log_code.ErrorConfigAuth, "invalid timeout '%s'", timeout)
		} else {
			auth.verify = veritification.NewCacheableVerify(timeout)
		}
	} else {
		auth.verify = veritification.NewRuntimeVerify()
	}
}

func Do(ctx *fasthttp.RequestCtx) (int64, error) {
	path := ctx.Path()
	uri := strings.Replace(strings.ToLower(string(path)), "/", "", -1) //todo await routing
	path = getPathWithoutPrefix(path)

	var (
		err   error
		appId int
	)

	if _, ok := routing.AddressMap[string(path)]; !ok {
		return 0, createError(codes.Unimplemented)
	}

	for _, notExpectedHeader := range notExpectedHeaders {
		ctx.Request.Header.Del(notExpectedHeader)
	}

	appToken := ctx.Request.Header.Peek(utils.ApplicationTokenHeader)
	if len(appToken) == 0 {
		return 0, createError(codes.Unauthenticated)
	} else if config.GetRemote().(*conf.RemoteConfig).TokenVerification.Enable {
		if appId, err = validateToken.Application(string(appToken)); err != nil {
			return 0, createError(codes.Unauthenticated, "application token parse")
		}
	}

	verifiableKeys, err := auth.verify.ApplicationToken(string(appToken))
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError(codes.Internal)
	}
	if len(verifiableKeys) != 4 {
		return 0, createError(codes.Unauthenticated, "application token value")
	}

	verifiableKeys[utils.DeviceTokenHeader] = string(ctx.Request.Header.Peek(utils.DeviceTokenHeader))
	verifiableKeys[utils.UserTokenHeader] = string(ctx.Request.Header.Peek(utils.UserTokenHeader))

	permittedToCall := false
	verifiableKeys, permittedToCall, err = auth.verify.Identity(verifiableKeys, uri)
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError(codes.Internal)
	}
	if permittedToCall {
		return 0, createError(codes.PermissionDenied)
	}

	applicationId, err := strconv.Atoi(verifiableKeys[utils.ApplicationIdHeader])
	if err != nil || (appId != 0 && applicationId != appId) {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError(codes.Internal)
	}

	for key, value := range verifiableKeys {
		if value != "" {
			ctx.Request.Header.Set(key, value)
		}
	}

	if _, ok := routing.InnerAddressMap[string(path)]; ok {
		adminToken := ctx.Request.Header.Peek("x-auth-admin") //todo const key
		if validateToken.Admin(string(adminToken)) != nil {
			return 0, createError(codes.PermissionDenied)
		}
	}

	return int64(applicationId), nil
}

func getPathWithoutPrefix(path []byte) []byte {
	firstFound := false
	for i, value := range path {
		if value == '/' {
			if firstFound {
				return path[i+1:]
			} else {
				firstFound = true
			}
		}
	}
	return []byte{}
}
