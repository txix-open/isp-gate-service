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

type UserAuthenticationCache struct {
	cache    *cache.Cache
	duration time.Duration
}

func NewUserAuthenticationCache(duration time.Duration) UserAuthenticationCache {
	return UserAuthenticationCache{
		duration: duration,
		cache:    cache.New(),
	}
}

func (r UserAuthenticationCache) Get(ctx context.Context, authModuleName string, token string) (*entity.UserAuthData, error) {
	data, ok := r.cache.Get(r.key(authModuleName, token))
	if !ok {
		return nil, domain.ErrAuthenticationCacheMiss
	}

	result := entity.UserAuthData{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, errors.WithMessage(err, "json unmarshal auth data")
	}

	return &result, nil
}

func (r UserAuthenticationCache) Set(
	ctx context.Context,
	authModuleName string,
	token string,
	data entity.UserAuthData,
) error {
	value, err := json.Marshal(data)
	if err != nil {
		return errors.WithMessage(err, "json marshal auth data")
	}

	r.cache.Set(r.key(authModuleName, token), value, r.duration)

	return nil
}

func (r UserAuthenticationCache) key(authModuleName string, token string) string {
	return fmt.Sprintf("%s:%s", authModuleName, token)
}
