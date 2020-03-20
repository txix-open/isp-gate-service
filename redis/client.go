package redis

import (
	rd "github.com/integration-system/isp-lib/v2/redis"
	log "github.com/integration-system/isp-log"
	"isp-gate-service/log_code"
)

var (
	Client = rd.NewRxClient(
		rd.WithInitHandler(func(c *rd.Client, err error) {
			if err != nil {
				log.Fatal(log_code.ErrorClientRedis, err)
			}
		}))
)
