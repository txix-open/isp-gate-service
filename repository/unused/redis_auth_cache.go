package unused

import (
	"context"
	"time"

	"isp-gate-service/domain"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"github.com/txix-open/isp-kit/json"
)

type RedisAuthCache struct {
	cli      redis.UniversalClient
	duration time.Duration
	db       int
}

func NewRedisAuthCache(cli redis.UniversalClient, duration time.Duration, db int) RedisAuthCache {
	return RedisAuthCache{
		cli:      cli,
		duration: duration,
		db:       db,
	}
}

func (r RedisAuthCache) Get(ctx context.Context, token string) (*domain.AuthData, error) {
	results, err := r.cli.Pipelined(ctx, func(p redis.Pipeliner) error {
		p.Select(ctx, r.db)
		p.Get(ctx, token)
		return nil
	})
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, errors.WithMessage(err, "pipelined")
	}

	data, err := results[1].(*redis.StringCmd).Result() // nolint: forcetypeassert
	if errors.Is(err, redis.Nil) {
		return nil, domain.ErrAuthenticationCacheMiss
	}
	if err != nil {
		return nil, errors.WithMessage(err, "get")
	}

	result := domain.AuthData{}
	err = json.Unmarshal([]byte(data), &result)
	if err != nil {
		return nil, errors.WithMessage(err, "json unmarshal auth data")
	}

	return &result, nil
}

func (r RedisAuthCache) Set(ctx context.Context, token string, data domain.AuthData) error {
	value, err := json.Marshal(data)
	if err != nil {
		return errors.WithMessage(err, "json marshal auth data")
	}

	results, err := r.cli.Pipelined(ctx, func(p redis.Pipeliner) error {
		p.Select(ctx, r.db)
		p.SetEx(ctx, token, string(value), r.duration)
		return nil
	})
	if err != nil {
		return errors.WithMessage(err, "pipelined")
	}

	err = results[1].Err()
	if err != nil {
		return errors.WithMessage(err, "set ex")
	}

	return nil
}
