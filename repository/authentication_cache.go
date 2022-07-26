package repository

import (
	"context"
	"time"

	"github.com/integration-system/isp-kit/json"
	"github.com/pkg/errors"
	"isp-gate-service/cache"
	"isp-gate-service/domain"
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

func (r AuthenticationCache) Get(ctx context.Context, token string) (*domain.AuthData, error) {
	data, ok := r.cache.Get(token)
	if !ok {
		return nil, domain.ErrAuthenticationCacheMiss
	}

	result := domain.AuthData{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, errors.WithMessage(err, "json unmarshal auth data")
	}

	return &result, nil
}

func (r AuthenticationCache) Set(ctx context.Context, token string, data domain.AuthData) error {
	value, err := json.Marshal(data)
	if err != nil {
		return errors.WithMessage(err, "json marshal auth data")
	}

	r.cache.Set(token, value, r.duration)

	return nil
}
