package authenticate

import (
	"github.com/integration-system/isp-lib/utils"
	log "github.com/integration-system/isp-log"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/log_code"
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
		return Error.create(codes.Unauthenticated)
	}

	keys, err := verification.appToken(string(appToken))
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return Error.create(codes.Internal)
	}
	if len(keys) != 4 {
		return Error.create(codes.Unauthenticated)
	}

	verifiableKeys := make(map[string]string)
	for i, value := range keys {
		verifiableKeys[verifiableKeyHeaderKeyMap[i]] = value
	}

	uri := strings.Replace(strings.ToLower(string(ctx.Path())), "/", "", -1)
	verifiableKeys[utils.DeviceTokenHeader] = string(ctx.Request.Header.Peek(utils.DeviceTokenHeader))
	verifiableKeys[utils.UserTokenHeader] = string(ctx.Request.Header.Peek(utils.UserTokenHeader))

	verifiableKeys, err = verification.keys(verifiableKeys, uri)
	if err != nil {
		log.Error(log_code.ErrorAuthenticate, err)
		return Error.create(codes.Internal)
	}
	if verifiableKeys[permittedToCallInfo] == "0" {
		return Error.create(codes.PermissionDenied)
	} else {
		delete(verifiableKeys, permittedToCallInfo)
	}

	for key, value := range verifiableKeys {
		if value != "" {
			ctx.Request.Header.Set(key, value)
		}
	}
	return nil
}
