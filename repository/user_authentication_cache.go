package repository

import (
	"context"
	"fmt"
	"time"

	"isp-gate-service/domain"
	"isp-gate-service/entity"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/json"
)

type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, data []byte, lifeTime time.Duration)
}

type UserAuthenticationCache struct {
	cache Cache
}

func NewUserAuthenticationCache(cache Cache) UserAuthenticationCache {
	return UserAuthenticationCache{
		cache: cache,
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
	duration time.Duration,
) error {
	value, err := json.Marshal(data)
	if err != nil {
		return errors.WithMessage(err, "json marshal auth data")
	}

	r.cache.Set(r.key(authModuleName, token), value, duration)

	return nil
}

func (r UserAuthenticationCache) key(authModuleName string, token string) string {
	return fmt.Sprintf("%s:%s", authModuleName, token)
}
