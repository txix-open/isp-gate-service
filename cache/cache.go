package cache

import (
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

func (c *Cache) now() time.Time {
	return time.Now()
}
