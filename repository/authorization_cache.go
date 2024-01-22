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

func (r AuthorizationCache) Get(ctx context.Context, id int, endpoint string) (bool, error) {
	key := r.key(id, endpoint)
	_, ok := r.cache.Get(key)
	return ok, nil
}

func (r AuthorizationCache) SetAuthorized(ctx context.Context, id int, endpoint string) error {
	key := r.key(id, endpoint)
	r.cache.Set(key, nil, r.duration)
	return nil
}

func (r AuthorizationCache) key(id int, endpoint string) string {
	return fmt.Sprintf("%d:%s", id, endpoint)
}
