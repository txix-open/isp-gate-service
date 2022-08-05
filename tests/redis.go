package tests

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/integration-system/isp-kit/test"
)

type Redis struct {
	address string
	redis.UniversalClient
}

func NewRedis(test *test.Test) Redis {
	redisHost := test.Config().Optional().String("REDIS_HOST", "localhost")
	redisPort := test.Config().Optional().String("REDIS_PORT", "6379")
	addr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	cli := redis.NewClient(&redis.Options{Addr: addr})
	return Redis{UniversalClient: cli, address: addr}
}

func (r Redis) Address() string {
	return r.address
}
