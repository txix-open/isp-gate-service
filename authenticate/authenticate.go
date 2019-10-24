package authenticate

import (
	"github.com/integration-system/isp-lib/utils"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"strings"
)

var (
	verifiableKeyHeaderKeyMap = map[string]string{
		SystemIdentity:      utils.SystemIdHeader,
		DomainIdentity:      utils.DomainIdHeader,
		ServiceIdentity:     utils.ServiceIdHeader,
		ApplicationIdentity: utils.ApplicationIdHeader,
		DeviceIdentity:      utils.DeviceIdHeader,
		UserIdentity:        utils.UserIdHeader,
		DeviceToken:         utils.DeviceTokenHeader,
		UserToken:           utils.UserTokenHeader,
	}
	notExpectedHeaders = []string{
		utils.SystemIdHeader, utils.DomainIdHeader, utils.ServiceIdHeader, utils.ApplicationIdHeader,
	}
)

func Do(ctx *fasthttp.RequestCtx) error {
	for _, notExpectedHeader := range notExpectedHeaders {
		ctx.Request.Header.Del(notExpectedHeader)
	}

	appToken := ctx.Request.Header.Peek(utils.ApplicationTokenHeader)
	if len(appToken) == 0 {
		return Error.Create(codes.Unauthenticated)
	}
	verifiableKeys, err := verification.appToken(string(appToken))
	if err != nil {
		return err
	}
	if len(verifiableKeys) != 4 {
		return Error.Create(codes.Unauthenticated)
	}

	uri := strings.Replace(strings.ToLower(string(ctx.Path())), "/", "", -1)
	verifiableKeys[DeviceToken] = string(ctx.Request.Header.Peek(utils.DeviceTokenHeader))
	verifiableKeys[UserToken] = string(ctx.Request.Header.Peek(utils.UserTokenHeader))

	verifiableKeys, err = verification.keys(verifiableKeys, uri)
	if err != nil {
		return err
	}

	for key, value := range verifiableKeys {
		if value != "" {
			ctx.Request.Header.Set(verifiableKeyHeaderKeyMap[key], value)
		}
	}
	return nil
}
