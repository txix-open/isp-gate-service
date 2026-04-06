package cache

import (
	"context"
	"sync"
	"time"
)

type Item struct {
	data      []byte
	expiredAt time.Time
}

type Cache struct {
	store map[string]Item
	lock  *sync.RWMutex
}

func New() *Cache {
	return &Cache{
		store: map[string]Item{},
		lock:  &sync.RWMutex{},
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	item, ok := c.store[key]
	if !ok {
		return nil, false
	}

	if c.now().After(item.expiredAt) {
		return nil, false
	}

	return item.data, true
}

func (c *Cache) Set(key string, data []byte, lifeTime time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.store[key] = Item{
		data:      data,
		expiredAt: c.now().Add(lifeTime),
	}
}

// StartCleaner runs periodic cleanup.
// Blocking call: intended to be run in a separate goroutine.
func (c *Cache) StartCleaner(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

func (c *Cache) cleanup() {
	c.lock.Lock()
	defer c.lock.Unlock()

	now := c.now()
	for k, v := range c.store {
		if now.After(v.expiredAt) {
			delete(c.store, k)
		}
	}
}

func (c *Cache) now() time.Time {
	return time.Now()
}
