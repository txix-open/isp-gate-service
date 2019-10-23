package redis

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/integration-system/isp-lib/config"
	rd "github.com/integration-system/isp-lib/redis"
	"github.com/integration-system/isp-lib/structure"
	"github.com/pkg/errors"
	"isp-gate-service/conf"
)

var Client = &redisClient{cli: rd.NewRxClient()}

type redisClient struct {
	cli *rd.RxClient
}

func (c *redisClient) ReceiveConfiguration(configuration structure.RedisConfiguration) {
	c.cli.ReceiveConfiguration(configuration)
}

func (c *redisClient) GetTokens(appToken string) (map[string]string, error) {
	if c.cli == nil {
		return nil, errors.New("client undefined")
	}

	instanceUuid := config.Get().(*conf.Configuration).InstanceUuid
	key := fmt.Sprintf("%s|%s", appToken, instanceUuid)

	if resp, err := c.cli.Pipelined(func(p redis.Pipeliner) error {
		if cmd := p.Select(int(rd.ApplicationTokenDb)); cmd.Err() != nil {
			return cmd.Err()
		}
		p.HGetAll(key)
		return nil
	}); err != nil {
		return nil, err
	} else {
		if len(resp) < 2 || resp[1].Err() == redis.Nil {
			return nil, nil
		} else if resp[1].Err() != nil {
			return nil, err
		} else if stringStringMapCmd, ok := resp[1].(*redis.StringStringMapCmd); !ok {
			return nil, nil
		} else {
			return stringStringMapCmd.Val(), nil
		}
	}
}

func (c *redisClient) CheckTokens(t map[string]string) (map[string]string, error) {
	secondDbKey := fmt.Sprintf("%s|%s", t[ApplicationIdentity], t[Uri])
	thirdDbKey := fmt.Sprintf("%s|%s", t[UserToken], t[DomainIdentity])
	fifthDbKey := fmt.Sprintf("%s|%s", t[DeviceToken], t[DomainIdentity])

	if resp, err := c.cli.Pipelined(func(p redis.Pipeliner) error {
		if cmd := p.Select(int(rd.ApplicationPermissionDb)); cmd.Err() != nil && cmd.Err() != redis.Nil {
			return cmd.Err()
		}
		p.Get(secondDbKey)

		if cmd := p.Select(int(rd.UserTokenDb)); cmd.Err() != nil && cmd.Err() != redis.Nil {
			return cmd.Err()
		}
		p.Get(thirdDbKey)

		if cmd := p.Select(int(rd.DeviceTokenDb)); cmd.Err() != nil && cmd.Err() != redis.Nil {
			return cmd.Err()
		}
		p.Get(fifthDbKey)
		return nil
	}); err != nil && err != redis.Nil {
		return t, err
	} else {
		// It is not permitted to call this methodParts
		if fineCmd(resp, 1) {
			if intCmd, ok := resp[1].(*redis.IntCmd); ok && intCmd.Val() == 0 {
				return t, errors.New("invalid token")
			}
		}
		//  ===== CHECK USER TOKEN =====
		if fineCmd(resp, 3) {
			if stringCmd, ok := resp[3].(*redis.StringCmd); ok {
				t[UserIdentity] = stringCmd.Val()
			}
		}
		// ===== CHECK DEVICE TOKEN =====
		if fineCmd(resp, 5) {
			if stringCmd, ok := resp[5].(*redis.StringCmd); ok {
				t[DeviceIdentity] = stringCmd.Val()
			}
		}
	}
	return t, nil
}

func fineCmd(cmders []redis.Cmder, arrayKey int) bool {
	return len(cmders) > arrayKey+1 && cmders[arrayKey] != nil &&
		(cmders[arrayKey].Err() == nil || cmders[arrayKey].Err() == redis.Nil)
}
