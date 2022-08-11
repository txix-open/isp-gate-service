package repository

import (
	"context"
	"fmt"
	"time"

	"isp-gate-service/cache"
)

type AuthorizationCache struct {
	duration time.Duration
	cache    *cache.Cache
}

func NewAuthorizationCache(duration time.Duration) *AuthorizationCache {
	return &AuthorizationCache{
		duration: duration,
		cache:    cache.New(),
	}
}

func (r AuthorizationCache) Get(ctx context.Context, applicationId int, endpoint string) (bool, error) {
	key := r.key(applicationId, endpoint)
	_, ok := r.cache.Get(key)
	return ok, nil
}

func (r AuthorizationCache) SetAuthorized(ctx context.Context, applicationId int, endpoint string) error {
	key := r.key(applicationId, endpoint)
	r.cache.Set(key, nil, r.duration)
	return nil
}

func (r AuthorizationCache) key(applicationId int, endpoint string) string {
	return fmt.Sprintf("%d:%s", applicationId, endpoint)
}