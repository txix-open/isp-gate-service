package authenticate

import (
	"github.com/integration-system/isp-lib/utils"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"isp-gate-service/redis"
	"strings"
)

var redisKeyHeaderKeyMap = map[string]string{
	redis.SystemIdentity:      utils.SystemIdHeader,
	redis.DomainIdentity:      utils.DomainIdHeader,
	redis.ServiceIdentity:     utils.ServiceIdHeader,
	redis.ApplicationIdentity: utils.ApplicationIdHeader,
	redis.DeviceIdentity:      utils.DeviceIdHeader,
	redis.UserIdentity:        utils.UserIdHeader,
	redis.DeviceToken:         utils.DeviceTokenHeader,
	redis.UserToken:           utils.UserTokenHeader,
}

func Compete(ctx *fasthttp.RequestCtx) error {
	for _, notExpectedHeader := range []string{
		utils.SystemIdHeader, utils.DomainIdHeader, utils.ServiceIdHeader, utils.ApplicationIdHeader,
	} {
		ctx.Request.Header.Del(notExpectedHeader)
	}

	appToken := ctx.Request.Header.Peek(utils.ApplicationTokenHeader)
	if len(appToken) == 0 {
		return errors.Errorf("%s not found", utils.ApplicationTokenHeader)
	}
	tokens, err := redis.Client.GetTokens(string(appToken))
	if err != nil || len(tokens) != 4 {
		return errors.Errorf("redis error %v", err)
	}

	tokens[redis.Uri] = strings.Replace(strings.ToLower(string(ctx.Path())), "/", "", -1)
	tokens[redis.DeviceToken] = string(ctx.Request.Header.Peek(utils.DeviceTokenHeader))
	tokens[redis.UserToken] = string(ctx.Request.Header.Peek(utils.UserTokenHeader))

	tokens, err = redis.Client.CheckTokens(tokens)
	if err != nil {
		return err
	}
	delete(tokens, redis.Uri)
	for key, value := range tokens {
		if value != "" {
			ctx.Request.Header.Set(redisKeyHeaderKeyMap[key], value)
		}
	}
	return nil
}
