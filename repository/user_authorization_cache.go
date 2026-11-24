package repository

import (
	"context"
	"fmt"
	"time"

	"isp-gate-service/cache"
)

type UserAuthorizationCache struct {
	duration time.Duration
	cache    *cache.Cache
}

func NewUserAuthorizationCache(duration time.Duration) *UserAuthorizationCache {
	return &UserAuthorizationCache{
		duration: duration,
		cache:    cache.New(),
	}
}

func (r UserAuthorizationCache) Get(ctx context.Context, authorizeEndpoint string, identity string, permission string) (bool, error) {
	key := r.key(authorizeEndpoint, identity, permission)
	_, ok := r.cache.Get(key)
	return ok, nil
}

func (r UserAuthorizationCache) SetAuthorized(ctx context.Context, authorizeEndpoint string, identity string, permission string) error {
	key := r.key(authorizeEndpoint, identity, permission)
	r.cache.Set(key, nil, r.duration)
	return nil
}

func (r UserAuthorizationCache) key(authorizeEndpoint string, identity string, permission string) string {
	return fmt.Sprintf("%s:%s:%s", authorizeEndpoint, identity, permission)
}
