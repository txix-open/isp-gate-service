package veritification

import (
	"sync"
	"time"
)

type (
	cacheableVerify struct {
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

func (cv *cacheableVerify) ApplicationToken(token string) (map[string]string, error) {
	cv.lock.RLock()
	cache, ok := cv.cache[token]
	cv.lock.RUnlock()
	if ok {
		if time.Now().Sub(cache.created) > cv.timeout {
			return cache.identity, nil
		}
	}

	cv.lock.Lock()
	defer cv.lock.Unlock()
	if cache, ok = cv.cache[token]; ok {
		if time.Now().Sub(cache.created) > cv.timeout {
			return cache.identity, nil
		} else {
			delete(cv.cache, token)
		}
	}
	if identity, err := cv.rv.ApplicationToken(token); err != nil {
		return nil, err
	} else {
		cv.cache[token] = cacheInfo{created: time.Now(), identity: identity}
		resp := identity
		return resp, nil
	}
}

func (cv *cacheableVerify) Identity(identity map[string]string, uri string) (map[string]string, bool, bool, error) {
	return cv.rv.Identity(identity, uri)
}
