package redis

import (
	rd "github.com/integration-system/isp-lib/v2/redis"
	"github.com/integration-system/isp-lib/v2/structure"
	log "github.com/integration-system/isp-log"
	"isp-gate-service/log_code"
)

var Client = &redisClient{
	cli: rd.NewRxClient(
		rd.WithInitHandler(func(c *rd.Client, err error) {
			if err != nil {
				log.Fatal(log_code.ErrorClientRedis, err)
			}
		})),
}

type redisClient struct {
	cli *rd.RxClient
}

func (c *redisClient) ReceiveConfiguration(configuration structure.RedisConfiguration) {
	c.cli.ReceiveConfiguration(configuration)
}

func (c *redisClient) Get() *rd.RxClient {
	return c.cli
}

func (c *redisClient) Close() {
	_ = c.cli.Close()
}
