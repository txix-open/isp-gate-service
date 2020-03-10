package veritification

import (
	rd "github.com/go-redis/redis"
	"github.com/integration-system/isp-lib/v2/config"
	"github.com/integration-system/isp-lib/v2/redis"
	"github.com/integration-system/isp-lib/v2/utils"
	"github.com/pkg/errors"
	"isp-gate-service/conf"
	rdClient "isp-gate-service/redis"
)

const permittedToCallInfo = "0"

var headerKeyByRedisIdentity = map[string]string{
	"1": utils.SystemIdHeader,
	"2": utils.DomainIdHeader,
	"3": utils.ServiceIdHeader,
	"4": utils.ApplicationIdHeader,
}

type runtimeVerify struct{}

func (v *runtimeVerify) ApplicationToken(token string) (map[string]string, error) {
	instanceUuid := config.Get().(*conf.Configuration).InstanceUuid
	key := makeDbKey(token, instanceUuid)

	if resp, err := rdClient.Client.Get().Pipelined(func(p rd.Pipeliner) error {
		if cmd := p.Select(int(redis.ApplicationTokenDb)); v.notEmptyError(cmd.Err()) {
			return cmd.Err()
		}
		if cmd := p.HGetAll(key); v.notEmptyError(cmd.Err()) {
			return cmd.Err()
		}
		return nil
	}); err != nil {
		return nil, err
	} else {
		if len(resp) < 2 || resp[1].Err() == rd.Nil {
			return nil, nil
		} else if resp[1].Err() != nil {
			return nil, err
		} else if stringStringMapCmd, ok := resp[1].(*rd.StringStringMapCmd); !ok {
			return nil, nil
		} else {
			vals := stringStringMapCmd.Val()
			identityMap := make(map[string]string, len(vals))
			for i, value := range vals {
				identityMap[headerKeyByRedisIdentity[i]] = value
			}
			return identityMap, nil
		}
	}
}

func (v *runtimeVerify) Identity(t map[string]string, uri string) (map[string]string, error) {
	secondDbKey := makeDbKey(t[utils.ApplicationIdHeader], uri)
	thirdDbKey := makeDbKey(t[utils.UserTokenHeader], t[utils.DomainIdHeader])
	fourthDbKey := makeDbKey(t[utils.UserIdHeader], uri)
	fifthDbKey := makeDbKey(t[utils.DeviceTokenHeader], t[utils.DomainIdHeader])

	if resp, err := rdClient.Client.Get().Pipelined(func(p rd.Pipeliner) error {
		if _, err := p.Select(int(redis.ApplicationPermissionDb)).Result(); v.notEmptyError(err) {
			return err
		}
		if _, err := p.Get(secondDbKey).Result(); v.notEmptyError(err) {
			return err
		}

		if _, err := p.Select(int(redis.UserTokenDb)).Result(); v.notEmptyError(err) {
			return err
		}
		if _, err := p.Get(thirdDbKey).Result(); v.notEmptyError(err) {
			return err
		}

		if _, err := p.Select(int(redis.UserPermissionDb)).Result(); v.notEmptyError(err) {
			return err
		}
		if _, err := p.Get(fourthDbKey).Result(); v.notEmptyError(err) {
			return err
		}

		if _, err := p.Select(int(redis.DeviceTokenDb)).Result(); v.notEmptyError(err) {
			return err
		}
		if _, err := p.Get(fifthDbKey).Result(); v.notEmptyError(err) {
			return err
		}

		return nil
	}); v.notEmptyError(err) {
		return t, err
	} else {
		// ===== NOT PERMITTED BY APPLICATION ID =====
		msg, err := v.findStringCmd(resp, 1)
		if err != nil {
			return t, err
		}
		if msg == permittedToCallInfo {
			return t, ErrorPermittedToCallApplication
		}
		// ===== CHECK USER TOKEN =====
		msg, err = v.findStringCmd(resp, 3)
		if err != nil {
			return t, err
		}
		userIdentity, found := t[utils.UserIdHeader]
		if found && msg != userIdentity {
			return t, ErrorInvalidUserId
		}
		// ===== NOT PERMITTED BY USER ID =====
		msg, err = v.findStringCmd(resp, 5)
		if err != nil {
			return t, err
		}
		if msg == permittedToCallInfo {
			return t, ErrorPermittedToCallUser
		}
		// ===== CHECK DEVICE TOKEN =====
		msg, err = v.findStringCmd(resp, 7)
		if err != nil {
			return t, err
		}
		t[utils.DeviceIdHeader] = msg
	}
	return t, nil
}

func (v *runtimeVerify) notEmptyError(err error) bool {
	return err != nil && err != rd.Nil
}

func (v *runtimeVerify) findStringCmd(cmders []rd.Cmder, arrayKey int) (string, error) {
	if len(cmders) > arrayKey {
		cmd := cmders[arrayKey]
		if cmd != nil {
			if cmd.Err() != nil {
				if cmd.Err() == rd.Nil {
					return "", nil
				}
				return "", cmd.Err()
			}
			if stringCmd, ok := cmd.(*rd.StringCmd); !ok {
				return "", errors.New("unexpected type")
			} else {
				return stringCmd.Val(), nil
			}
		} else {
			return "", errors.New("empty cmd")
		}
	} else {
		return "", errors.New("not found cmd")
	}
}

func makeDbKey(key, val string) string {
	return key + "|" + val
}
