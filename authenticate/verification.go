package authenticate

import (
	"fmt"
	rd "github.com/go-redis/redis"
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/redis"
	"github.com/integration-system/isp-lib/utils"
	"github.com/pkg/errors"
	"isp-gate-service/conf"
	rdClient "isp-gate-service/redis"
)

const (
	systemIdentity      = "1"
	domainIdentity      = "2"
	serviceIdentity     = "3"
	applicationIdentity = "4"

	permittedToCallInfo = "permittedToCallInfo"
)

var verification verificationHelper

type verificationHelper struct{}

func (v verificationHelper) appToken(token string) (map[string]string, error) {
	instanceUuid := config.Get().(*conf.Configuration).InstanceUuid
	key := fmt.Sprintf("%s|%s", token, instanceUuid)

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
			return stringStringMapCmd.Val(), nil
		}
	}
}

func (v *verificationHelper) keys(t map[string]string, uri string) (map[string]string, error) {
	secondDbKey := fmt.Sprintf("%s|%s", t[utils.ApplicationIdHeader], uri)
	thirdDbKey := fmt.Sprintf("%s|%s", t[utils.UserTokenHeader], t[utils.DomainIdHeader])
	fifthDbKey := fmt.Sprintf("%s|%s", t[utils.DeviceTokenHeader], t[utils.DomainIdHeader])

	if resp, err := rdClient.Client.Get().Pipelined(func(p rd.Pipeliner) error {
		if cmd := p.Select(int(redis.ApplicationPermissionDb)); v.notEmptyError(cmd.Err()) {
			return cmd.Err()
		}
		if cmd := p.Get(secondDbKey); v.notEmptyError(cmd.Err()) {
			return cmd.Err()
		}

		if cmd := p.Select(int(redis.UserTokenDb)); v.notEmptyError(cmd.Err()) {
			return cmd.Err()
		}
		if cmd := p.Get(thirdDbKey); v.notEmptyError(cmd.Err()) {
			return cmd.Err()
		}

		if cmd := p.Select(int(redis.DeviceTokenDb)); v.notEmptyError(cmd.Err()) {
			return cmd.Err()
		}
		if cmd := p.Get(fifthDbKey); v.notEmptyError(cmd.Err()) {
			return cmd.Err()
		}
		return nil
	}); v.notEmptyError(err) {
		return t, err
	} else {
		// It is not permitted to call this methodParts
		if msg, err := v.findStringCmd(resp, 1); err != nil {
			return t, err
		} else {
			t[permittedToCallInfo] = msg
		}
		//  ===== CHECK USER TOKEN =====
		if msg, err := v.findStringCmd(resp, 3); err != nil {
			return t, err
		} else {
			t[utils.UserIdHeader] = msg
		}
		// ===== CHECK DEVICE TOKEN =====
		if msg, err := v.findStringCmd(resp, 5); err != nil {
			return t, err
		} else {
			t[utils.DeviceIdHeader] = msg
		}
	}
	return t, nil
}

func (v *verificationHelper) notEmptyError(err error) bool {
	return err != nil && err != rd.Nil
}

func (v *verificationHelper) findStringCmd(cmders []rd.Cmder, arrayKey int) (string, error) {
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
