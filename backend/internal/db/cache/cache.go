package cache

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned by Get when the key does not exist.
var ErrCacheMiss = errors.New("cache miss")

// Client is a minimal key-value cache abstraction over Redis or an in-memory store.
type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	// Incr atomically increments an integer counter (creates with value 1 if absent).
	Incr(ctx context.Context, key string) (int64, error)
	// Expire updates the TTL of an existing key.
	Expire(ctx context.Context, key string, ttl time.Duration) error
}
