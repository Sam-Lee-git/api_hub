package cache

import (
	"context"
	"sync"
	"time"
)

type memEntry struct {
	value   string
	counter int64
	expiry  time.Time
}

func (e *memEntry) expired() bool {
	return !e.expiry.IsZero() && time.Now().After(e.expiry)
}

type memCache struct {
	mu   sync.Mutex
	data map[string]*memEntry
}

// NewMemoryCache returns an in-process TTL cache implementing Client.
// Suitable for single-instance deployments (e.g. SQLite mode on a small server).
func NewMemoryCache() Client {
	c := &memCache{data: make(map[string]*memEntry)}
	go c.evict()
	return c
}

func (c *memCache) evict() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		for k, e := range c.data {
			if e.expired() {
				delete(c.data, k)
			}
		}
		c.mu.Unlock()
	}
}

func (c *memCache) Get(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.data[key]
	if !ok || e.expired() {
		return "", ErrCacheMiss
	}
	return e.value, nil
}

func (c *memCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var expiry time.Time
	if ttl > 0 {
		expiry = time.Now().Add(ttl)
	}
	c.data[key] = &memEntry{value: value, expiry: expiry}
	return nil
}

func (c *memCache) Del(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

func (c *memCache) Incr(_ context.Context, key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.data[key]
	if !ok || e.expired() {
		c.data[key] = &memEntry{counter: 1}
		return 1, nil
	}
	e.counter++
	return e.counter, nil
}

func (c *memCache) Expire(_ context.Context, key string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.data[key]
	if !ok {
		return nil
	}
	e.expiry = time.Now().Add(ttl)
	return nil
}
