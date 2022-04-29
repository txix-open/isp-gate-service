package veritification

import (
	"context"

	rd "github.com/go-redis/redis/v8"
	"github.com/integration-system/isp-lib/v2/config"
	"github.com/integration-system/isp-lib/v2/redis"
	"github.com/integration-system/isp-lib/v2/utils"
	"github.com/pkg/errors"
	"isp-gate-service/conf"
	rdClient "isp-gate-service/redis"
)

const (
	permittedToCallInfo = "0"

	appIdArrayKey       = 1
	userTokenArrayKey   = 3
	userIdArrayKey      = 5
	deviceTokenArrayKey = 7
)

var (
	errNoRedisCmd            = errors.New("no redis cmd")
	errUnexpectedCmd         = errors.New("expected StringCmd")
	headerKeyByRedisIdentity = map[string]string{
		"1": utils.SystemIdHeader,
		"2": utils.DomainIdHeader,
		"3": utils.ServiceIdHeader,
		"4": utils.ApplicationIdHeader,
	}
)

type runtimeVerify struct{}

func (v *runtimeVerify) ApplicationToken(token string) (map[string]string, error) {
	instanceUuid := config.Get().(*conf.Configuration).InstanceUuid
	key := makeDbKey(token, instanceUuid)

	resp, err := rdClient.Client.UseDb(redis.ApplicationTokenDb, func(p rd.Pipeliner) error {
		if cmd := p.HGetAll(context.Background(), key); v.isError(cmd.Err()) {
			return cmd.Err()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(resp) < 2 || resp[1].Err() == rd.Nil {
		return nil, nil
	} else if resp[1].Err() != nil {
		return nil, err
	} else if stringStringMapCmd, ok := resp[1].(*rd.StringStringMapCmd); !ok {
		return nil, errUnexpectedCmd
	} else {
		vals := stringStringMapCmd.Val()
		identityMap := make(map[string]string, len(vals))
		for i, value := range vals {
			identityMap[headerKeyByRedisIdentity[i]] = value
		}
		return identityMap, nil
	}
}

func (v *runtimeVerify) Identity(t map[string]string, uri string) (map[string]string, error) {
	secondDbKey := makeDbKey(t[utils.ApplicationIdHeader], uri)
	thirdDbKey := makeDbKey(t[utils.UserTokenHeader], t[utils.DomainIdHeader])
	fourthDbKey := makeDbKey(t[utils.UserIdHeader], uri)
	fifthDbKey := makeDbKey(t[utils.DeviceTokenHeader], t[utils.DomainIdHeader])

	resp, err := rdClient.Client.Pipelined(context.Background(), func(p rd.Pipeliner) error {
		if _, err := p.Select(context.Background(), int(redis.ApplicationPermissionDb)).Result(); v.isError(err) {
			return err
		}
		if _, err := p.Get(context.Background(), secondDbKey).Result(); v.isError(err) {
			return err
		}

		if _, err := p.Select(context.Background(), int(redis.UserTokenDb)).Result(); v.isError(err) {
			return err
		}
		if _, err := p.Get(context.Background(), thirdDbKey).Result(); v.isError(err) {
			return err
		}

		if _, err := p.Select(context.Background(), int(redis.UserPermissionDb)).Result(); v.isError(err) {
			return err
		}
		if _, err := p.Get(context.Background(), fourthDbKey).Result(); v.isError(err) {
			return err
		}

		if _, err := p.Select(context.Background(), int(redis.DeviceTokenDb)).Result(); v.isError(err) {
			return err
		}
		if _, err := p.Get(context.Background(), fifthDbKey).Result(); v.isError(err) {
			return err
		}

		return nil
	})
	if v.isError(err) {
		return t, err
	}
	// ===== NOT PERMITTED BY APPLICATION ID =====
	msg, err := v.findStringCmd(resp, appIdArrayKey)
	if err != nil {
		return t, err
	}
	if msg == permittedToCallInfo {
		return t, ErrorPermittedToCallApplication
	}
	// ===== CHECK USER TOKEN =====
	msg, err = v.findStringCmd(resp, userTokenArrayKey)
	if err != nil {
		return t, err
	}
	userIdentity, found := t[utils.UserIdHeader]
	if found && msg != userIdentity {
		return t, ErrorInvalidUserId
	}
	// ===== NOT PERMITTED BY USER ID =====
	msg, err = v.findStringCmd(resp, userIdArrayKey)
	if err != nil {
		return t, err
	}
	if msg == permittedToCallInfo {
		return t, ErrorPermittedToCallUser
	}
	// ===== CHECK DEVICE TOKEN =====
	msg, err = v.findStringCmd(resp, deviceTokenArrayKey)
	if err != nil {
		return t, err
	}
	t[utils.DeviceIdHeader] = msg
	return t, nil
}

func (v *runtimeVerify) isError(err error) bool {
	return err != nil && err != rd.Nil
}

func (v *runtimeVerify) findStringCmd(cmders []rd.Cmder, arrayKey int) (string, error) {
	if len(cmders) <= arrayKey {
		return "", errNoRedisCmd
	}

	cmd := cmders[arrayKey]
	if cmd == nil {
		return "", errNoRedisCmd
	}

	if v.isError(cmd.Err()) {
		return "", cmd.Err()
	}

	if stringCmd, ok := cmd.(*rd.StringCmd); !ok {
		return "", errUnexpectedCmd
	} else {
		return stringCmd.Val(), nil
	}
}

func makeDbKey(key, val string) string {
	return key + "|" + val
}
