package authenticate

import (
	"github.com/integration-system/isp-lib/utils"
	log "github.com/integration-system/isp-log"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/log_code"
	"isp-gate-service/routing"
	"strconv"
	"strings"
)

var (
	verifiableKeyHeaderKeyMap = map[string]string{
		systemIdentity:      utils.SystemIdHeader,
		domainIdentity:      utils.DomainIdHeader,
		serviceIdentity:     utils.ServiceIdHeader,
		applicationIdentity: utils.ApplicationIdHeader,
	}
)

func Do(ctx *fasthttp.RequestCtx) (int64, error) {
	for _, notExpectedHeader := range verifiableKeyHeaderKeyMap {
		ctx.Request.Header.Del(notExpectedHeader)
	}

	appToken := ctx.Request.Header.Peek(utils.ApplicationTokenHeader)
	if len(appToken) == 0 {
		return 0, createError(codes.Unauthenticated)
	}

	keys, err := verification.appToken(string(appToken))
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError(codes.Internal)
	}
	if len(keys) != 4 {
		return 0, createError(codes.Unauthenticated)
	}

	verifiableKeys := make(map[string]string)
	for i, value := range keys {
		verifiableKeys[verifiableKeyHeaderKeyMap[i]] = value
	}

	path := ctx.Path()
	uri := strings.Replace(strings.ToLower(string(path)), "/", "", -1)
	verifiableKeys[utils.DeviceTokenHeader] = string(ctx.Request.Header.Peek(utils.DeviceTokenHeader))
	verifiableKeys[utils.UserTokenHeader] = string(ctx.Request.Header.Peek(utils.UserTokenHeader))

	verifiableKeys, err = verification.keys(verifiableKeys, uri)
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError(codes.Internal)
	}
	if verifiableKeys[permittedToCallInfo] == "0" {
		return 0, createError(codes.PermissionDenied)
	} else {
		delete(verifiableKeys, permittedToCallInfo)
	}

	applicationId, err := strconv.Atoi(verifiableKeys[utils.ApplicationIdHeader])
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return 0, createError(codes.Internal)
	}

	for key, value := range verifiableKeys {
		if value != "" {
			ctx.Request.Header.Set(key, value)
		}
	}

	path = getPathWithoutPrefix(path)
	if _, ok := routing.InnerAddressMap[string(path)]; ok {
		adminToken := ctx.Request.Header.Peek("x-auth-admin") //todo const key
		if token.Check(string(adminToken)) != nil {
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
