package repository

import (
	"context"
	"fmt"
	"time"

	"isp-gate-service/cache"
	"isp-gate-service/domain"
	"isp-gate-service/entity"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/json"
)

type CustomAuthenticationCache struct {
	cache    *cache.Cache
	duration time.Duration
}

func NewCustomAuthenticationCache(duration time.Duration) CustomAuthenticationCache {
	return CustomAuthenticationCache{
		duration: duration,
		cache:    cache.New(),
	}
}

func (r CustomAuthenticationCache) Get(ctx context.Context, authName string, token string) (*entity.CustomAuthenticateResponse, error) {
	data, ok := r.cache.Get(token)
	if !ok {
		return nil, domain.ErrAuthenticationCacheMiss
	}

	result := entity.CustomAuthenticateResponse{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, errors.WithMessage(err, "json unmarshal auth data")
	}

	return &result, nil
}

func (r CustomAuthenticationCache) Set(ctx context.Context, authName string, token string, data entity.CustomAuthenticateResponse) error {
	value, err := json.Marshal(data)
	if err != nil {
		return errors.WithMessage(err, "json marshal auth data")
	}

	r.cache.Set(r.cacheKey(authName, token), value, r.duration)

	return nil
}

func (r CustomAuthenticationCache) cacheKey(authName string, token string) string {
	return fmt.Sprintf("%s;%s", authName, token)
}
