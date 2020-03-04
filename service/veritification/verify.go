package veritification

import (
	"time"
)

var Default = NewRuntimeVerify()

type Verify interface {
	ApplicationToken(string) (map[string]string, error)
	Identity(map[string]string, string) (map[string]string, error)
}

func NewRuntimeVerify() Verify {
	return &runtimeVerify{}
}

func NewCacheablesVerify(timeout time.Duration) Verify {
	return &cacheablesVerify{
		cache:   make(map[string]cacheInfo),
		timeout: timeout,
		rv:      &runtimeVerify{},
	}
}
