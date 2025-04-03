package unused

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type RedisAuthzCache struct {
	cli      redis.UniversalClient
	duration time.Duration
	db       int
}

func NewRedisAuthzCache(cli redis.UniversalClient, duration time.Duration, db int) *RedisAuthzCache {
	return &RedisAuthzCache{
		cli:      cli,
		duration: duration,
		db:       db,
	}
}

func (r RedisAuthzCache) Get(ctx context.Context, applicationId int, endpoint string) (bool, error) {
	result, err := r.cli.Pipelined(ctx, func(p redis.Pipeliner) error {
		p.Select(ctx, r.db)
		p.Get(ctx, r.key(applicationId, endpoint))
		return nil
	})
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, errors.WithMessage(err, "pipelined")
	}

	_, err = result[1].(*redis.StringCmd).Result() // nolint: forcetypeassert
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithMessage(err, "get")
	}
	return true, nil
}

func (r RedisAuthzCache) SetAuthorized(ctx context.Context, applicationId int, endpoint string) error {
	result, err := r.cli.Pipelined(ctx, func(p redis.Pipeliner) error {
		p.Select(ctx, r.db)
		p.SetEx(ctx, r.key(applicationId, endpoint), "authorized", r.duration)
		return nil
	})
	if err != nil {
		return errors.WithMessage(err, "pipelined")
	}

	err = result[1].Err()
	if err != nil {
		return errors.WithMessage(err, "set ex")
	}

	return nil
}

func (r RedisAuthzCache) key(applicationId int, endpoint string) string {
	return fmt.Sprintf("%d:%s", applicationId, endpoint)
}
