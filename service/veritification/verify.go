package veritification

import (
	"time"
)

var Default = NewRuntimeVerify()

type Verify interface {
	ApplicationToken(string) (map[string]string, error)
	Identity(map[string]string, string) (map[string]string, bool, bool, error)
}

func NewRuntimeVerify() Verify {
	return &runtimeVerify{}
}

func NewCacheableVerify(timeout time.Duration) Verify {
	return &cacheableVerify{
		cache:   make(map[string]cacheInfo),
		timeout: timeout,
		rv:      &runtimeVerify{},
	}
}
