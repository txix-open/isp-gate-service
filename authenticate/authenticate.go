package authenticate

import (
	"github.com/integration-system/isp-lib/utils"
	log "github.com/integration-system/isp-log"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/log_code"
	"isp-gate-service/routing"
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

func Do(ctx *fasthttp.RequestCtx) error {
	for _, notExpectedHeader := range verifiableKeyHeaderKeyMap {
		ctx.Request.Header.Del(notExpectedHeader)
	}

	appToken := ctx.Request.Header.Peek(utils.ApplicationTokenHeader)
	if len(appToken) == 0 {
		return createError(codes.Unauthenticated)
	}

	keys, err := verification.appToken(string(appToken))
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return createError(codes.Internal)
	}
	if len(keys) != 4 {
		return createError(codes.Unauthenticated)
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
		return createError(codes.Internal)
	}
	if verifiableKeys[permittedToCallInfo] == "0" {
		return createError(codes.PermissionDenied)
	} else {
		delete(verifiableKeys, permittedToCallInfo)
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
			return createError(codes.PermissionDenied)
		}
	}
	return nil
}

func getPathWithoutPrefix(path []byte) []byte {
	firstFound := false
	for key, value := range path {
		if value == '/' {
			if firstFound {
				return path[key+1:]
			} else {
				firstFound = true
			}
		}
	}
	return []byte{}
}
