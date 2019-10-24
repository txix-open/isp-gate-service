package authenticate

import (
	"fmt"
	rd "github.com/go-redis/redis"
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/redis"
	"google.golang.org/grpc/codes"
	"isp-gate-service/conf"
	rdClient "isp-gate-service/redis"
)

const (
	SystemIdentity      = "1"
	DomainIdentity      = "2"
	ServiceIdentity     = "3"
	ApplicationIdentity = "4"

	DeviceIdentity = "5"
	UserIdentity   = "6"

	DeviceToken = "7"
	UserToken   = "8"
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
		return nil, Error.Create(codes.Internal)
	} else {
		if len(resp) < 2 || resp[1].Err() == rd.Nil {
			return nil, nil
		} else if resp[1].Err() != nil {
			return nil, Error.Create(codes.Unauthenticated)
		} else if stringStringMapCmd, ok := resp[1].(*rd.StringStringMapCmd); !ok {
			return nil, nil
		} else {
			return stringStringMapCmd.Val(), nil
		}
	}
}

func (v *verificationHelper) keys(t map[string]string, uri string) (map[string]string, error) {
	secondDbKey := fmt.Sprintf("%s|%s", t[ApplicationIdentity], uri)
	thirdDbKey := fmt.Sprintf("%s|%s", t[UserToken], t[DomainIdentity])
	fifthDbKey := fmt.Sprintf("%s|%s", t[DeviceToken], t[DomainIdentity])

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
	}); err != nil && err != rd.Nil {
		return t, Error.Create(codes.Internal)
	} else {
		// It is not permitted to call this methodParts
		if v.fineCmd(resp, 1) {
			if intCmd, ok := resp[1].(*rd.IntCmd); ok && intCmd.Val() == 0 {
				return t, Error.Create(codes.PermissionDenied)
			}
		}
		//  ===== CHECK USER TOKEN =====
		if v.fineCmd(resp, 3) {
			if stringCmd, ok := resp[3].(*rd.StringCmd); ok {
				t[UserIdentity] = stringCmd.Val()
			}
		}
		// ===== CHECK DEVICE TOKEN =====
		if v.fineCmd(resp, 5) {
			if stringCmd, ok := resp[5].(*rd.StringCmd); ok {
				t[DeviceIdentity] = stringCmd.Val()
			}
		}
	}
	return t, nil
}

func (v *verificationHelper) notEmptyError(err error) bool {
	return err != nil && err != rd.Nil
}

func (v *verificationHelper) fineCmd(cmders []rd.Cmder, arrayKey int) bool {
	if len(cmders) > arrayKey+1 {
		cmder := cmders[arrayKey]
		return cmder != nil && (cmder.Err() == nil || cmder.Err() == rd.Nil)
	} else {
		return false
	}
}
