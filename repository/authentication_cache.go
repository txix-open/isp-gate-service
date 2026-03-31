package repository

import (
	"context"
	"time"

	"isp-gate-service/cache"
	"isp-gate-service/domain"
	"isp-gate-service/entity"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/json"
)

type AuthenticationCache struct {
	cache    *cache.Cache
	duration time.Duration
}

func NewAuthenticationCache(duration time.Duration) AuthenticationCache {
	return AuthenticationCache{
		duration: duration,
		cache:    cache.New(),
	}
}

func (r AuthenticationCache) Get(ctx context.Context, token string) (*entity.AuthData, error) {
	data, ok := r.cache.Get(token)
	if !ok {
		return nil, domain.ErrAuthenticationCacheMiss
	}

	result := entity.AuthData{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, errors.WithMessage(err, "json unmarshal auth data")
	}

	return &result, nil
}

func (r AuthenticationCache) Set(ctx context.Context, token string, data entity.AuthData) error {
	value, err := json.Marshal(data)
	if err != nil {
		return errors.WithMessage(err, "json marshal auth data")
	}

	r.cache.Set(token, value, r.duration)

	return nil
}
