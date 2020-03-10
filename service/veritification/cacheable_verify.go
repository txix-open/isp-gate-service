package veritification

import (
	"sync"
	"time"
)

type (
	cacheablesVerify struct {
		cache   map[string]cacheInfo
		timeout time.Duration
		rv      *runtimeVerify
		lock    sync.RWMutex
	}

	cacheInfo struct {
		created  time.Time
		identity map[string]string
	}
)

func (cv *cacheablesVerify) ApplicationToken(token string) (map[string]string, error) {
	cv.lock.RLock()
	cache, ok := cv.cache[token]
	cv.lock.RUnlock()
	if ok {
		if time.Since(cache.created) < cv.timeout {
			return cv.copyCache(cache.identity), nil
		}
	}

	cv.lock.Lock()
	defer cv.lock.Unlock()
	cache, ok = cv.cache[token]
	if ok {
		if time.Since(cache.created) < cv.timeout {
			return cv.copyCache(cache.identity), nil
		}
		delete(cv.cache, token)
	}

	identity, err := cv.rv.ApplicationToken(token)
	if err != nil {
		return nil, err
	}
	cv.cache[token] = cacheInfo{created: time.Now(), identity: identity}
	return cv.copyCache(identity), nil
}

func (cv *cacheablesVerify) Identity(identity map[string]string, uri string) (map[string]string, error) {
	return cv.rv.Identity(identity, uri)
}

func (cv *cacheablesVerify) copyCache(cache map[string]string) map[string]string {
	resp := make(map[string]string, 4)
	for key, value := range cache {
		resp[key] = value
	}
	return resp
}
